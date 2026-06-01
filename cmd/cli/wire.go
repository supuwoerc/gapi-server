//go:build wireinject
// +build wireinject

package main

import (
	"github.com/supuwoerc/gapi-server/internal/app"
	"github.com/supuwoerc/gapi-server/internal/provider"

	"github.com/google/wire"
)

func WireCli() (*app.Cli, error) {
	wire.Build(
		provider.ConfigSet,
		provider.BaseInfraSet,
		wire.Struct(new(app.Cli), "*"),
	)
	return nil, nil
}
