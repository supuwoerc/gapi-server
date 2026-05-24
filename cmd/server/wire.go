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
		provider.HandlerSet,
		provider.ServerSet,
		app.NewApp,
	)
	return nil, nil
}
