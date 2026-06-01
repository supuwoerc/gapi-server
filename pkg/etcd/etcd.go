package etcd

import (
	"context"
	"time"

	"github.com/supuwoerc/gapi-server/internal/config"

	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type Logger interface {
	Info(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

func NewClient(cfg *config.EtcdConfig, l Logger) (*clientv3.Client, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: time.Duration(cfg.DialTimeout) * time.Second,
		Username:    cfg.Username,
		Password:    cfg.Password,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create etcd client")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.DialTimeout)*time.Second)
	defer cancel()
	_, err = client.Status(ctx, cfg.Endpoints[0])
	if err != nil {
		_ = client.Close()
		return nil, errors.Wrap(err, "failed to connect to etcd")
	}
	l.Info("etcd connected", zap.Strings("endpoints", cfg.Endpoints))
	return client, nil
}
