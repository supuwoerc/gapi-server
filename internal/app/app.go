package app

import (
	"gapi-server/internal/server"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type App struct {
	server *server.HttpServer
	logger *zap.Logger
	db     *gorm.DB
}

func NewApp(server *server.HttpServer, logger *zap.Logger, db *gorm.DB) *App {
	return &App{server: server, logger: logger, db: db}
}

func (a *App) Run() {
	a.server.Run()
}

func (a *App) Close() {
	defer func() {
		_ = a.logger.Sync()
	}()
	defer a.logger.Info("app clean is executed")
	if sqlDB, err := a.db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
