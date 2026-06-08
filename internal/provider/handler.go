package provider

import (
	v1 "github.com/supuwoerc/gapi-server/internal/handler/v1"
	"github.com/supuwoerc/gapi-server/internal/router"

	"github.com/google/wire"
)

var HandlerSet = wire.NewSet(
	wire.Struct(new(v1.HealthHandler), "*"),
	ProvideV1Registrars,
	router.NewV1Handlers,
)

func ProvideV1Registrars(health *v1.HealthHandler, cronJob *v1.CronJobHandler, auth *v1.AuthHandler, captchaH *v1.CaptchaHandler) []router.Registrar {
	return []router.Registrar{health, cronJob, auth, captchaH}
}
