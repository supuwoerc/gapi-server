package provider

import (
	"github.com/supuwoerc/gapi-server/internal/handler/v1"
	"github.com/supuwoerc/gapi-server/internal/router"

	"github.com/google/wire"
)

var HandlerSet = wire.NewSet(
	v1.NewHealthHandler,
	ProvideV1Registrars,
	router.NewV1Handlers,
)

func ProvideV1Registrars(health *v1.HealthHandler) []router.Registrar {
	return []router.Registrar{health}
}
