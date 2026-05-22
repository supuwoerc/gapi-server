//go:build wireinject
// +build wireinject

package main

import (
	"gapi-server/internal/config"
	"gapi-server/internal/handler"
	"gapi-server/internal/repository"
	"gapi-server/internal/router"
	"gapi-server/internal/service"
	"gapi-server/pkg/database"
	"gapi-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

func WireApp() (*gin.Engine, func(), error) {
	wire.Build(
		config.NewViper,
		config.NewConfig,
		provideLogConfig,
		provideDBConfig,
		provideServerConfig,

		logger.NewZapLogger,
		database.NewConnection,

		repository.NewRepository,

		service.NewService,

		handler.NewHealthHandler,

		router.NewEngine,
	)
	return nil, nil, nil
}

func provideLogConfig(cfg *config.Config) *config.LogConfig {
	return &cfg.Log
}

func provideDBConfig(cfg *config.Config) *config.DatabaseConfig {
	return &cfg.Database
}

func provideServerConfig(cfg *config.Config) *config.ServerConfig {
	return &cfg.Server
}
