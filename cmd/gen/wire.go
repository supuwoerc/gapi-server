//go:build wireinject
// +build wireinject

package main

import (
	"github.com/supuwoerc/gapi-server/internal/app"
	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/internal/provider"
	"github.com/supuwoerc/gapi-server/pkg/database"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/google/wire"
)

func WireGen() (*app.Gen, error) {
	wire.Build(
		config.NewViper,
		config.NewConfig,
		provider.ProvideLogConfig,
		provider.ProvideDBConfig,
		logger.NewLogger,
		database.NewConnection,
		wire.Struct(new(app.Gen), "*"),
	)
	return nil, nil
}
