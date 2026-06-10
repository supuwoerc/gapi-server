//go:build wireinject
// +build wireinject

package main

import (
	"github.com/supuwoerc/gapi-server/internal/app"
	"github.com/supuwoerc/gapi-server/internal/provider"

	"github.com/google/wire"
)

func WireApp() (*app.App, error) {
	wire.Build(
		provider.ConfigSet,
		provider.InfraSet,
		provider.CronJobSet,
		provider.CaptchaSet,
		provider.EmailSet,
		provider.AuthSet,
		provider.HandlerSet,
		provider.ServerSet,
		wire.Struct(new(app.App), "*"),
	)
	return nil, nil
}
