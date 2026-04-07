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

func WireApp(cfg *config.Config) (*gin.Engine, func(), error) {
	wire.Build(
		// Config providers
		provideLogConfig,
		provideDBConfig,

		// Infrastructure
		logger.Init,
		database.Init,

		// Data layer
		repository.NewRepository,

		// Business layer
		service.NewService,

		// Handler layer
		handler.NewHealthHandler,

		// Router
		router.Init,
	)
	return nil, nil, nil
}

func provideLogConfig(cfg *config.Config) *config.LogConfig {
	return &cfg.Log
}

func provideDBConfig(cfg *config.Config) *config.DatabaseConfig {
	return &cfg.Database
}
