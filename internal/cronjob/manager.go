package cronjob

import (
	"context"
	"runtime/debug"
	"sync"
	"time"

	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type JobRecorder interface {
	SyncJobDefinitions(ctx context.Context, jobs []SystemJob) error
	IsJobEnabled(ctx context.Context, name string) (bool, error)
	RecordStart(ctx context.Context, jobName, triggeredBy string) (uint64, error)
	RecordEnd(ctx context.Context, executionID uint64, status string, jobErr error) error
	UpdateLastRun(ctx context.Context, name string, status string) error
}

type JobManager struct {
	cron      *cron.Cron
	logger    *logger.Logger
	recorder  JobRecorder
	cfg       *config.CronConfig
	jobs      []SystemJob
	entryMap  map[string]cron.EntryID
	cancelMap map[string]context.CancelFunc
	mu        sync.RWMutex
}

func NewJobManager(l *logger.Logger, recorder JobRecorder, cfg *config.CronConfig, jobs []SystemJob) *JobManager {
	return &JobManager{
		logger:    l,
		recorder:  recorder,
		cfg:       cfg,
		jobs:      jobs,
		entryMap:  make(map[string]cron.EntryID),
		cancelMap: make(map[string]context.CancelFunc),
	}
}

func (m *JobManager) Start(ctx context.Context) error {
	if !m.cfg.Enabled {
		m.logger.Info("cron: scheduler disabled by config")
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
			m.logger.Info("cron: job disabled, skipping", zap.String("job", j.Name()))
			continue
		}
		if err := m.registerJob(j); err != nil {
			return errors.Wrapf(err, "cron: register job %s", j.Name())
		}
	}

	m.cron.Start()
	m.logger.Info("cron: scheduler started", zap.Int("registered_jobs", len(m.entryMap)))
	return nil
}

func (m *JobManager) Stop() {
	if m.cron == nil {
		return
	}
	m.logger.Info("cron: scheduler stopping...")

	stopCtx := m.cron.Stop()

	m.mu.RLock()
	for name, cancel := range m.cancelMap {
		m.logger.Info("cron: cancelling running job", zap.String("job", name))
		cancel()
	}
	m.mu.RUnlock()

	timeout := time.Duration(m.cfg.ShutdownTimeout) * time.Second
	select {
	case <-stopCtx.Done():
		m.logger.Info("cron: all jobs finished")
	case <-time.After(timeout):
		m.logger.Warn("cron: shutdown timeout reached, some jobs may not have finished")
	}
}

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
			go m.executeWithRecording(ctx, j, TriggerByManual)
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
			return m.registerJob(j)
		}
	}
	return errors.Errorf("job not found: %s", jobName)
}

func (m *JobManager) DisableJob(_ context.Context, jobName string, cancelRunning bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	entryID, exists := m.entryMap[jobName]
	if exists {
		m.cron.Remove(entryID)
		delete(m.entryMap, jobName)
		m.logger.Info("cron: job removed from scheduler", zap.String("job", jobName))
	}
	if cancelRunning {
		if cancel, ok := m.cancelMap[jobName]; ok {
			cancel()
			m.logger.Info("cron: cancelled running job", zap.String("job", jobName))
		}
	}
	return nil
}

func (m *JobManager) Jobs() []SystemJob {
	return m.jobs
}

func (m *JobManager) registerJob(j SystemJob) error {
	wrappedJob := m.wrapJob(j)
	id, err := m.cron.AddJob(j.Interval(), wrappedJob)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.entryMap[j.Name()] = id
	m.mu.Unlock()
	m.logger.Info("cron: job registered",
		zap.String("name", j.Name()),
		zap.String("interval", j.Interval()),
	)
	return nil
}

func (m *JobManager) wrapJob(j SystemJob) cron.Job {
	handler := cron.FuncJob(func() {
		ctx, cancel := context.WithCancel(context.Background())
		m.mu.Lock()
		m.cancelMap[j.Name()] = cancel
		m.mu.Unlock()
		defer func() {
			cancel()
			m.mu.Lock()
			delete(m.cancelMap, j.Name())
			m.mu.Unlock()
		}()
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

func (m *JobManager) executeWithRecording(ctx context.Context, j SystemJob, triggeredBy string) {
	execID, err := m.recorder.RecordStart(ctx, j.Name(), triggeredBy)
	if err != nil {
		m.logger.Error("cron: failed to record job start", zap.String("job", j.Name()), zap.Error(err))
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
				m.logger.Error("cron: job panicked",
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
		if recordErr := m.recorder.RecordEnd(ctx, execID, status, jobErr); recordErr != nil {
			m.logger.Error("cron: failed to record job end", zap.String("job", j.Name()), zap.Error(recordErr))
		}
	}

	if updateErr := m.recorder.UpdateLastRun(ctx, j.Name(), status); updateErr != nil {
		m.logger.Error("cron: failed to update last run", zap.String("job", j.Name()), zap.Error(updateErr))
	}

	duration := time.Since(startTime)
	m.logger.Info("cron: job completed",
		zap.String("job", j.Name()),
		zap.String("status", status),
		zap.Duration("duration", duration),
	)
}
