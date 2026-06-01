package app

import (
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Cli struct {
	Logger *logger.Logger
	DB     *gorm.DB
	Redis  *redis.Client
	Etcd   *clientv3.Client
}

func (c *Cli) Close() {
	defer func() {
		_ = c.Logger.Sync()
	}()
	defer c.Logger.Info("cli clean is executed")
	if sqlDB, err := c.DB.DB(); err != nil {
		c.Logger.Error("failed to get sql.DB", zap.Error(err))
	} else if err := sqlDB.Close(); err != nil {
		c.Logger.Error("failed to close database", zap.Error(err))
	}
	if err := c.Redis.Close(); err != nil {
		c.Logger.Error("failed to close redis", zap.Error(err))
	}
	if err := c.Etcd.Close(); err != nil {
		c.Logger.Error("failed to close etcd", zap.Error(err))
	}
}
