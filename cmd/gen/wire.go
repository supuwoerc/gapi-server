//go:build wireinject
// +build wireinject

package main

import (
	"github.com/supuwoerc/gapi-server/internal/app"
	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/pkg/database"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/google/wire"
)

func WireGen() (*app.Gen, error) {
	wire.Build(
		config.NewViper,
		config.NewConfig,
		provideLogConfig,
		provideDBConfig,
		logger.NewLogger,
		database.NewConnection,
		wire.Struct(new(app.Gen), "*"),
	)
	return nil, nil
}

func provideLogConfig(cfg *config.Config) *config.LogConfig {
	return &cfg.Log
}

func provideDBConfig(cfg *config.Config) *config.DatabaseConfig {
	return &cfg.Database
}
