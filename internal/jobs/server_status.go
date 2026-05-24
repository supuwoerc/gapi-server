package jobs

import (
	"context"
	"runtime"

	"github.com/supuwoerc/gapi-server/internal/cronjob"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"go.uber.org/zap"
)

var _ cronjob.SystemJob = (*ServerStatusJob)(nil)

type ServerStatusJob struct {
	logger *logger.Logger
}

func NewServerStatusJob(l *logger.Logger) *ServerStatusJob {
	return &ServerStatusJob{logger: l}
}

func (j *ServerStatusJob) Name() string {
	return "server_status"
}

func (j *ServerStatusJob) Interval() string {
	return "0 */5 * * * *"
}

func (j *ServerStatusJob) ExecutionMode() cronjob.ExecutionMode {
	return cronjob.ModeSkipIfRunning
}

func (j *ServerStatusJob) Handle(_ context.Context) error {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	j.logger.Info("server status",
		zap.Int("goroutines", runtime.NumGoroutine()),
		zap.Uint64("alloc_mb", mem.Alloc/1024/1024),
		zap.Uint64("sys_mb", mem.Sys/1024/1024),
	)
	return nil
}
