package app

import (
	"context"

	"github.com/supuwoerc/gapi-server/internal/cronjob"
	"github.com/supuwoerc/gapi-server/internal/server"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type App struct {
	server     *server.HttpServer
	logger     *logger.Logger
	db         *gorm.DB
	redis      *redis.Client
	etcd       *clientv3.Client
	jobManager *cronjob.JobManager
}

func NewApp(server *server.HttpServer, logger *logger.Logger, db *gorm.DB, redis *redis.Client, etcd *clientv3.Client, jobManager *cronjob.JobManager) *App {
	return &App{server: server, logger: logger, db: db, redis: redis, etcd: etcd, jobManager: jobManager}
}

func (a *App) Run() {
	if err := a.jobManager.Start(context.Background()); err != nil {
		a.logger.Fatal("failed to start job manager", zap.Error(err))
	}
	a.server.Run()
}

func (a *App) Close() {
	defer func() {
		_ = a.logger.Sync()
	}()
	defer a.logger.Info("app clean is executed")
	a.jobManager.Stop()
	if sqlDB, err := a.db.DB(); err != nil {
		a.logger.Error("failed to get sql.DB", zap.Error(err))
	} else if err := sqlDB.Close(); err != nil {
		a.logger.Error("failed to close database", zap.Error(err))
	}
	if err := a.redis.Close(); err != nil {
		a.logger.Error("failed to close redis", zap.Error(err))
	}
	if err := a.etcd.Close(); err != nil {
		a.logger.Error("failed to close etcd", zap.Error(err))
	}
}
