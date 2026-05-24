package provider

import (
	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/internal/cronjob"
	v1 "github.com/supuwoerc/gapi-server/internal/handler/v1"
	"github.com/supuwoerc/gapi-server/internal/jobs"
	"github.com/supuwoerc/gapi-server/internal/repository"
	"github.com/supuwoerc/gapi-server/internal/service"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/google/wire"
)

var CronJobSet = wire.NewSet(
	ProvideCronConfig,
	ProvideSystemJobs,
	repository.NewCronJobRepository,
	service.NewCronJobService,
	wire.Bind(new(cronjob.JobRecorder), new(*service.CronJobService)),
	cronjob.NewJobManager,
	v1.NewCronJobHandler,
)

func ProvideCronConfig(cfg *config.Config) *config.CronConfig {
	return &cfg.Cron
}

func ProvideSystemJobs(l *logger.Logger) []cronjob.SystemJob {
	return []cronjob.SystemJob{
		jobs.NewServerStatusJob(l),
	}
}
