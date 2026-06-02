package provider

import (
	"github.com/supuwoerc/gapi-server/internal/cronjob"
	"github.com/supuwoerc/gapi-server/internal/dal"
	v1 "github.com/supuwoerc/gapi-server/internal/handler/v1"
	"github.com/supuwoerc/gapi-server/internal/jobs"
	"github.com/supuwoerc/gapi-server/internal/service"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/google/wire"
)

var CronJobSet = wire.NewSet(
	ProvideSystemJobs,
	cronjob.NewLockerAdapter,
	wire.Bind(new(cronjob.DistLocker), new(*cronjob.LockerAdapter)),
	wire.Struct(new(dal.CronJobDal), "*"),
	wire.Struct(new(service.CronJobService), "*"),
	wire.Bind(new(service.CronJobRepository), new(*dal.CronJobDal)),
	wire.Bind(new(cronjob.JobRecorder), new(*service.CronJobService)),
	wire.Bind(new(v1.CronJobService), new(*service.CronJobService)),
	cronjob.NewJobManager,
	wire.Struct(new(v1.CronJobHandler), "*"),
)

func ProvideSystemJobs(l *logger.Logger) []cronjob.SystemJob {
	return []cronjob.SystemJob{
		jobs.NewServerStatusJob(l),
	}
}
