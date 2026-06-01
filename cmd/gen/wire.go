//go:build wireinject
// +build wireinject

package main

import (
	"github.com/supuwoerc/gapi-server/internal/app"
	"github.com/supuwoerc/gapi-server/internal/provider"

	"github.com/google/wire"
)

func WireGen() (*app.Gen, error) {
	wire.Build(
		provider.ConfigSet,
		provider.BaseInfraSet,
		wire.Struct(new(app.Gen), "*"),
	)
	return nil, nil
}
