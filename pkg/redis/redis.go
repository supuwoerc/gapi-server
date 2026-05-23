package redis

import (
	"context"
	"fmt"

	"gapi-server/internal/config"
	"gapi-server/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func NewClient(cfg *config.RedisConfig, l *logger.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	client.AddHook(NewHook(l, LogLevel(cfg.LogLevel)))
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	l.Info("redis connected", zap.String("addr", cfg.Addr), zap.Int("db", cfg.DB))
	return client, nil
}
