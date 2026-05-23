package app

import (
	"github.com/supuwoerc/gapi-server/internal/server"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type App struct {
	server *server.HttpServer
	logger *logger.Logger
	db     *gorm.DB
	redis  *redis.Client
}

func NewApp(server *server.HttpServer, logger *logger.Logger, db *gorm.DB, redis *redis.Client) *App {
	return &App{server: server, logger: logger, db: db, redis: redis}
}

func (a *App) Run() {
	a.server.Run()
}

func (a *App) Close() {
	defer func() {
		_ = a.logger.Sync()
	}()
	defer a.logger.Info("app clean is executed")
	if sqlDB, err := a.db.DB(); err != nil {
		a.logger.Error("failed to get sql.DB", zap.Error(err))
	} else if err := sqlDB.Close(); err != nil {
		a.logger.Error("failed to close database", zap.Error(err))
	}
	if err := a.redis.Close(); err != nil {
		a.logger.Error("failed to close redis", zap.Error(err))
	}
}
