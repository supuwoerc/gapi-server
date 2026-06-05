package app

import (
	"github.com/supuwoerc/gapi-server/internal/server"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type App struct {
	Server *server.HttpServer
	Logger *logger.Logger
	DB     *gorm.DB
	Redis  *redis.Client
	Etcd   *clientv3.Client
}

func (a *App) Run() {
	a.Server.Run()
}

func (a *App) Close() {
	defer func() {
		_ = a.Logger.Sync()
	}()
	defer a.Logger.Info("app clean is executed")
	if db, err := a.DB.DB(); err != nil {
		a.Logger.Error("failed to get app sql.DB", zap.Error(err))
	} else if err := db.Close(); err != nil {
		a.Logger.Error("failed to close app database", zap.Error(err))
	}
	if err := a.Redis.Close(); err != nil {
		a.Logger.Error("failed to close redis", zap.Error(err))
	}
	if err := a.Etcd.Close(); err != nil {
		a.Logger.Error("failed to close etcd", zap.Error(err))
	}
}
