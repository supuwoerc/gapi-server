package cronjob

import (
	"context"
	"testing"
	"time"

	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// tracingJob 在 Handle 内部用 ctx 打日志, 模拟真实 job 的行为。
// started/release 非 nil 时用于构造确定性时序。
type tracingJob struct {
	l       Logger
	started chan struct{}
	release chan struct{}
	done    chan struct{}
}

func (j *tracingJob) Name() string                 { return "trace-verify" }
func (j *tracingJob) Interval() string             { return "* * * * * *" }
func (j *tracingJob) ExecutionMode() ExecutionMode { return ModeSkipIfRunning }

func (j *tracingJob) Handle(ctx context.Context) error {
	j.l.Ctx(ctx).Info("inside job handle")
	closeOnce(j.started)
	if j.release != nil {
		<-j.release
	}
	closeOnce(j.done)
	return nil
}

func closeOnce(ch chan struct{}) {
	if ch == nil {
		return
	}
	select {
	case <-ch:
	default:
		close(ch)
	}
}

func fieldOf(e observer.LoggedEntry, key string) string {
	for _, f := range e.Context {
		if f.Key == key {
			return f.String
		}
	}
	return ""
}

func traceIDOf(e observer.LoggedEntry) string {
	return fieldOf(e, string(logger.TraceIDKey))
}

type TraceSuite struct {
	suite.Suite
}

func (s *TraceSuite) newRecordingManager(job SystemJob, enabled bool) (*JobManager, *observer.ObservedLogs, Logger) {
	core, logs := observer.New(zap.DebugLevel)
	l := &mockLogger{Logger: zap.New(core)}
	cfg := &config.CronConfig{Enabled: enabled, ShutdownTimeout: 5}
	return NewJobManager(l, &mockRecorder{}, cfg, []SystemJob{job}, nil), logs, l
}

// 调度执行: job 内部日志与 manager 收尾日志共用同一个 trace id。
func (s *TraceSuite) TestScheduledRunSharesTraceID() {
	job := &tracingJob{done: make(chan struct{})}
	mgr, logs, l := s.newRecordingManager(job, true)
	job.l = l

	s.Require().NoError(mgr.Start(context.Background()))
	select {
	case <-job.done:
	case <-time.After(3 * time.Second):
		s.FailNow("job never ran")
	}
	time.Sleep(300 * time.Millisecond)
	mgr.Stop(context.Background())

	inside := logs.FilterMessage("inside job handle").All()
	completed := logs.FilterMessage("cron: job completed").All()
	s.Require().NotEmpty(inside, "job handle log missing")
	s.Require().NotEmpty(completed, "job completed log missing")

	insideTrace := traceIDOf(inside[0])
	s.NotEmpty(insideTrace, "job handle log has no trace_id")
	s.Equal(insideTrace, traceIDOf(completed[0]), "trace_id not shared within one execution")

	// 启动期日志走 Start 的 ctx, 不应带上执行期的 trace id
	for _, e := range logs.FilterMessage("cron: job registered").All() {
		s.Empty(traceIDOf(e), "registration log should not carry execution trace_id")
	}
}

// 手动触发: 请求 ctx 的 trace id 透传, 且 handler 返回(请求 ctx 取消)后 job 不应被判定为 cancelled。
func (s *TraceSuite) TestManualTriggerPropagatesTraceIDAndSurvivesRequestCancel() {
	job := &tracingJob{
		started: make(chan struct{}),
		release: make(chan struct{}),
		done:    make(chan struct{}),
	}
	mgr, logs, l := s.newRecordingManager(job, false)
	job.l = l

	reqCtx, cancel := context.WithCancel(logger.WithTraceID(context.Background(), "req-trace-123"))
	s.Require().NoError(mgr.TriggerManual(reqCtx, job.Name(), false))

	// 时序: 等 job 进入 Handle -> 取消请求 ctx(模拟 handler 返回) -> 放行 job
	select {
	case <-job.started:
	case <-time.After(3 * time.Second):
		s.FailNow("manual job never started")
	}
	cancel()
	time.Sleep(50 * time.Millisecond)
	close(job.release)

	select {
	case <-job.done:
	case <-time.After(3 * time.Second):
		s.FailNow("manual job never finished")
	}
	time.Sleep(300 * time.Millisecond)

	completed := logs.FilterMessage("cron: job completed").All()
	s.Require().NotEmpty(completed)
	s.Equal("req-trace-123", traceIDOf(completed[0]), "request trace_id not propagated to job logs")
	s.Equal(StatusSuccess, fieldOf(completed[0], "status"),
		"job should not be marked cancelled just because the request ctx was cancelled")
}

func TestTraceSuite(t *testing.T) {
	suite.Run(t, new(TraceSuite))
}
