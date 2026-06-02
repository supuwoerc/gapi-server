package cronjob

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/pkg/etcd"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type mockLockResult struct{}

func (m *mockLockResult) Unlock(_ context.Context) error { return nil }

type mockLocker struct {
	shouldFail bool
}

func (m *mockLocker) TryLock(_ context.Context, _ string) (LockResult, error) {
	if m.shouldFail {
		return nil, etcd.ErrLockNotAcquired
	}
	return &mockLockResult{}, nil
}

type mockRecorder struct{}

func (m *mockRecorder) SyncJobDefinitions(_ context.Context, _ []SystemJob) error      { return nil }
func (m *mockRecorder) IsJobEnabled(_ context.Context, _ string) (bool, error)         { return true, nil }
func (m *mockRecorder) RecordStart(_ context.Context, _, _ string) (uint64, error)     { return 1, nil }
func (m *mockRecorder) RecordEnd(_ context.Context, _ uint64, _ string, _ error) error { return nil }
func (m *mockRecorder) UpdateLastRun(_ context.Context, _ string, _ string) error      { return nil }

type countingJob struct {
	name  string
	count atomic.Int64
}

func (j *countingJob) Name() string                 { return j.name }
func (j *countingJob) Interval() string             { return "* * * * * *" }
func (j *countingJob) ExecutionMode() ExecutionMode { return ModeSkipIfRunning }
func (j *countingJob) Handle(_ context.Context) error {
	j.count.Add(1)
	return nil
}

type ManagerSuite struct {
	suite.Suite
}

func (s *ManagerSuite) newManager(locker DistLocker, job *countingJob) *JobManager {
	l, _ := zap.NewDevelopment()
	cfg := &config.CronConfig{Enabled: true, ShutdownTimeout: 5}
	return NewJobManager(l, &mockRecorder{}, cfg, []SystemJob{job}, locker)
}

func (s *ManagerSuite) TestExecutesWhenLockAcquired() {
	job := &countingJob{name: "test-lock-acquired"}
	mgr := s.newManager(&mockLocker{shouldFail: false}, job)

	err := mgr.Start(context.Background())
	s.Require().NoError(err)

	time.Sleep(2 * time.Second)
	mgr.Stop()

	s.Greater(job.count.Load(), int64(0))
}

func (s *ManagerSuite) TestSkipsWhenLockFailed() {
	job := &countingJob{name: "test-lock-failed"}
	mgr := s.newManager(&mockLocker{shouldFail: true}, job)

	err := mgr.Start(context.Background())
	s.Require().NoError(err)

	time.Sleep(2 * time.Second)
	mgr.Stop()

	s.Equal(int64(0), job.count.Load())
}

func (s *ManagerSuite) TestNilLockerExecutesNormally() {
	job := &countingJob{name: "test-nil-locker"}
	mgr := s.newManager(nil, job)

	err := mgr.Start(context.Background())
	s.Require().NoError(err)

	time.Sleep(2 * time.Second)
	mgr.Stop()

	s.Greater(job.count.Load(), int64(0))
}

func TestManagerSuite(t *testing.T) {
	suite.Run(t, new(ManagerSuite))
}
