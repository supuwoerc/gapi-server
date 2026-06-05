package app

import (
	"context"

	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ICliHook interface {
	OnInit(ctx context.Context) error
	OnClose(ctx context.Context) error
}

type BaseCliHook struct{}

func (BaseCliHook) OnInit(context.Context) error  { return nil }
func (BaseCliHook) OnClose(context.Context) error { return nil }

type Cli struct {
	Logger *logger.Logger
	DB     *gorm.DB
	Redis  *redis.Client
	Etcd   *clientv3.Client
	Hooks  []ICliHook
}

func (c *Cli) Init() {
	for _, h := range c.Hooks {
		if err := h.OnInit(context.Background()); err != nil {
			c.Logger.Fatal("cli hook OnInit failed", zap.Error(err))
		}
	}
}

func (c *Cli) Close() {
	defer func() {
		_ = c.Logger.Sync()
	}()
	defer c.Logger.Info("cli clean is executed")
	for i := len(c.Hooks) - 1; i >= 0; i-- {
		if err := c.Hooks[i].OnClose(context.Background()); err != nil {
			c.Logger.Error("cli hook OnClose failed", zap.Error(err))
		}
	}
	if db, err := c.DB.DB(); err != nil {
		c.Logger.Error("failed to get cli sql.DB", zap.Error(err))
	} else if err := db.Close(); err != nil {
		c.Logger.Error("failed to close cli database", zap.Error(err))
	}
	if err := c.Redis.Close(); err != nil {
		c.Logger.Error("failed to close redis", zap.Error(err))
	}
	if err := c.Etcd.Close(); err != nil {
		c.Logger.Error("failed to close etcd", zap.Error(err))
	}
}
