//go:build wireinject
// +build wireinject

package main

import (
	"gapi-server/internal/app"
	"gapi-server/internal/provider"

	"github.com/google/wire"
)

func WireApp() (*app.App, error) {
	wire.Build(
		provider.ConfigSet,
		provider.InfraSet,
		provider.HandlerSet,
		provider.ServerSet,
		app.NewApp,
	)
	return nil, nil
}
