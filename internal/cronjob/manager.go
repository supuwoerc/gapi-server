package cronjob

import (
	"context"
	"runtime/debug"
	"sync"
	"time"

	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/internal/dal/model"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type JobRecorder interface {
	SyncJobDefinitions(ctx context.Context, jobs []SystemJob) error
	IsJobEnabled(ctx context.Context, name string) (bool, error)
	RecordStart(ctx context.Context, jobName string, triggeredBy model.TriggeredBy) (int64, error)
	RecordEnd(ctx context.Context, executionID int64, status string, jobErr error) error
	UpdateLastRun(ctx context.Context, name string, status string) error
}

type JobManager struct {
	cron      *cron.Cron
	logger    Logger
	recorder  JobRecorder
	cfg       *config.CronConfig
	jobs      []SystemJob
	entryMap  map[string]cron.EntryID
	cancelMap map[string]context.CancelFunc
	mu        sync.RWMutex
	locker    DistLocker
}

func NewJobManager(l Logger, recorder JobRecorder, cfg *config.CronConfig, jobs []SystemJob, locker DistLocker) *JobManager {
	return &JobManager{
		logger:    l,
		recorder:  recorder,
		cfg:       cfg,
		jobs:      jobs,
		entryMap:  make(map[string]cron.EntryID),
		cancelMap: make(map[string]context.CancelFunc),
		locker:    locker,
	}
}

func (m *JobManager) Start(ctx context.Context) error {
	if !m.cfg.Enabled {
		m.logger.Ctx(ctx).Info("cron: scheduler disabled by config")
		return nil
	}

	cronLogger := NewCronLogger(m.logger)
	m.cron = cron.New(
		cron.WithSeconds(),
		cron.WithLogger(cronLogger),
		cron.WithChain(cron.Recover(cronLogger)),
	)

	if err := m.recorder.SyncJobDefinitions(ctx, m.jobs); err != nil {
		return errors.Wrap(err, "cron: sync job definitions")
	}

	for _, j := range m.jobs {
		enabled, err := m.recorder.IsJobEnabled(ctx, j.Name())
		if err != nil {
			return errors.Wrapf(err, "cron: check job enabled %s", j.Name())
		}
		if !enabled {
			m.logger.Ctx(ctx).Info("cron: job disabled, skipping", zap.String("job", j.Name()))
			continue
		}
		if err := m.registerJob(ctx, j); err != nil {
			return errors.Wrapf(err, "cron: register job %s", j.Name())
		}
	}

	m.cron.Start()
	m.logger.Ctx(ctx).Info("cron: scheduler started", zap.Int("registered_jobs", len(m.entryMap)))
	return nil
}

func (m *JobManager) Stop(ctx context.Context) {
	if m.cron == nil {
		return
	}
	log := m.logger.Ctx(ctx)
	log.Info("cron: scheduler stopping...")

	stopCtx := m.cron.Stop()

	m.mu.RLock()
	for name, cancel := range m.cancelMap {
		log.Info("cron: cancelling running job", zap.String("job", name))
		cancel()
	}
	m.mu.RUnlock()

	timeout := time.Duration(m.cfg.ShutdownTimeout) * time.Second
	select {
	case <-stopCtx.Done():
		log.Info("cron: all jobs finished")
	case <-time.After(timeout):
		log.Warn("cron: shutdown timeout reached, some jobs may not have finished")
	}
}

func (m *JobManager) OnStart(ctx context.Context) error { return m.Start(ctx) }
func (m *JobManager) OnReady(context.Context) error     { return nil }
func (m *JobManager) OnStop(ctx context.Context) error  { m.Stop(ctx); return nil }

func (m *JobManager) TriggerManual(ctx context.Context, jobName string, force bool) error {
	for _, j := range m.jobs {
		if j.Name() == jobName {
			if !force {
				m.mu.RLock()
				_, running := m.cancelMap[jobName]
				m.mu.RUnlock()
				if running && j.ExecutionMode() != ModeAllowConcurrent {
					return ErrJobRunning
				}
			}
			// job 在独立 goroutine 中执行, 会比 HTTP 请求活得更久。这里剥离请求 ctx 的
			// 取消信号(否则 handler 返回后 job 立刻被判定为 cancelled), 但保留 trace id。
			jobCtx := context.WithoutCancel(ctx)
			go m.executeWithRecording(jobCtx, j, TriggerByManual)
			return nil
		}
	}
	return errors.Errorf("job not found: %s", jobName)
}

func (m *JobManager) EnableJob(ctx context.Context, jobName string) error {
	for _, j := range m.jobs {
		if j.Name() == jobName {
			m.mu.RLock()
			_, exists := m.entryMap[jobName]
			m.mu.RUnlock()
			if exists {
				return nil
			}
			return m.registerJob(ctx, j)
		}
	}
	return errors.Errorf("job not found: %s", jobName)
}

func (m *JobManager) DisableJob(ctx context.Context, jobName string, cancelRunning bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	log := m.logger.Ctx(ctx)
	entryID, exists := m.entryMap[jobName]
	if exists {
		m.cron.Remove(entryID)
		delete(m.entryMap, jobName)
		log.Info("cron: job removed from scheduler", zap.String("job", jobName))
	}
	if cancelRunning {
		if cancel, ok := m.cancelMap[jobName]; ok {
			cancel()
			log.Info("cron: cancelled running job", zap.String("job", jobName))
		}
	}
	return nil
}

func (m *JobManager) Jobs() []SystemJob {
	return m.jobs
}

func (m *JobManager) registerJob(ctx context.Context, j SystemJob) error {
	wrappedJob := m.wrapJob(j)
	id, err := m.cron.AddJob(j.Interval(), wrappedJob)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.entryMap[j.Name()] = id
	m.mu.Unlock()
	m.logger.Ctx(ctx).Info("cron: job registered",
		zap.String("name", j.Name()),
		zap.String("interval", j.Interval()),
	)
	return nil
}

func (m *JobManager) wrapJob(j SystemJob) cron.Job {
	handler := cron.FuncJob(func() {
		// 每次调度生成独立的 trace id, 便于串联单次执行产生的所有日志。
		base := logger.WithTraceID(context.Background(), logger.GenerateTraceID())
		ctx, cancel := context.WithCancel(base)
		m.mu.Lock()
		m.cancelMap[j.Name()] = cancel
		m.mu.Unlock()
		defer func() {
			cancel()
			m.mu.Lock()
			delete(m.cancelMap, j.Name())
			m.mu.Unlock()
		}()

		if m.locker != nil {
			lock, err := m.locker.TryLock(ctx, j.Name())
			if err != nil {
				m.logger.Ctx(ctx).Debug("cron: skipping job, another instance is running",
					zap.String("job", j.Name()))
				return
			}
			defer func() { _ = lock.Unlock(ctx) }()
		}

		m.executeWithRecording(ctx, j, TriggerByScheduler)
	})

	cronLogger := NewCronLogger(m.logger)
	switch j.ExecutionMode() {
	case ModeSkipIfRunning:
		return cron.NewChain(cron.SkipIfStillRunning(cronLogger)).Then(handler)
	case ModeDelayIfRunning:
		return cron.NewChain(cron.DelayIfStillRunning(cronLogger)).Then(handler)
	default:
		return handler
	}
}

func (m *JobManager) executeWithRecording(ctx context.Context, j SystemJob, triggeredBy model.TriggeredBy) {
	// 保证 ctx 中一定有 trace id, 单次执行的所有日志(含 job 内部)共用同一个 id。
	if logger.TraceIDFromContext(ctx) == "" {
		ctx = logger.WithTraceID(ctx, logger.GenerateTraceID())
	}
	log := m.logger.Ctx(ctx)
	// 收尾的落库不受 job ctx 取消影响, 否则 job 被取消时最终状态写不进去。
	recordCtx := context.WithoutCancel(ctx)

	execID, err := m.recorder.RecordStart(recordCtx, j.Name(), triggeredBy)
	if err != nil {
		log.Error("cron: failed to record job start", zap.String("job", j.Name()), zap.Error(err))
	}

	startTime := time.Now()
	var jobErr error
	var status string

	func() {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				jobErr = errors.Errorf("panic: %v\n%s", r, stack)
				status = StatusPanic
				log.Error("cron: job panicked",
					zap.String("job", j.Name()),
					zap.Any("panic", r),
					zap.String("stack", stack),
				)
			}
		}()
		jobErr = j.Handle(ctx)
	}()

	if status == "" {
		if ctx.Err() != nil {
			status = StatusCancelled
		} else if jobErr != nil {
			status = StatusFailed
		} else {
			status = StatusSuccess
		}
	}

	if execID > 0 {
		if recordErr := m.recorder.RecordEnd(recordCtx, execID, status, jobErr); recordErr != nil {
			log.Error("cron: failed to record job end", zap.String("job", j.Name()), zap.Error(recordErr))
		}
	}

	if updateErr := m.recorder.UpdateLastRun(recordCtx, j.Name(), status); updateErr != nil {
		log.Error("cron: failed to update last run", zap.String("job", j.Name()), zap.Error(updateErr))
	}

	duration := time.Since(startTime)
	log.Info("cron: job completed",
		zap.String("job", j.Name()),
		zap.String("status", status),
		zap.Duration("duration", duration),
	)
}
