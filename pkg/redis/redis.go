package redis

import (
	"context"
	"fmt"

	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
	"go.uber.org/zap"
)

func NewClient(cfg *config.RedisConfig, l *logger.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		MaintNotificationsConfig: &maintnotifications.Config{
			Mode: maintnotifications.Mode(cfg.MaintNotifications),
		},
	})
	client.AddHook(NewHook(l, LogLevel(cfg.LogLevel)))
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	l.Info("redis connected", zap.String("addr", cfg.Addr), zap.Int("db", cfg.DB))
	return client, nil
}
