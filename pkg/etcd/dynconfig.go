package etcd

import (
	"bytes"
	"context"
	"sync"

	"github.com/supuwoerc/gapi-server/internal/config"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type DynConfig struct {
	client *clientv3.Client
	cfg    *config.EtcdConfig
	v      *viper.Viper
	appCfg *config.Config
	logger Logger
	key    string
	mu     sync.RWMutex
	cancel context.CancelFunc
	done   chan struct{}
}

func NewDynConfig(client *clientv3.Client, cfg *config.EtcdConfig, v *viper.Viper, appCfg *config.Config, l Logger) *DynConfig {
	key := cfg.DynConfig.Key
	if key == "" {
		key = "/gapi/config/app.yaml"
	}
	return &DynConfig{
		client: client,
		cfg:    cfg,
		v:      v,
		appCfg: appCfg,
		logger: l,
		key:    key,
		done:   make(chan struct{}),
	}
}

func (d *DynConfig) Start(ctx context.Context) error {
	if !d.cfg.DynConfig.Enabled {
		d.logger.Info("etcd dynconfig: disabled by config")
		return nil
	}

	resp, err := d.client.Get(ctx, d.key)
	if err != nil {
		return errors.Wrap(err, "etcd dynconfig: get remote config")
	}

	if len(resp.Kvs) > 0 {
		value := resp.Kvs[0].Value
		if err := d.v.MergeConfig(bytes.NewReader(value)); err != nil {
			return errors.Wrap(err, "etcd dynconfig: merge remote config")
		}
		if err := d.v.Unmarshal(d.appCfg); err != nil {
			return errors.Wrap(err, "etcd dynconfig: unmarshal after merge")
		}
		d.logger.Info("etcd dynconfig: remote config loaded and merged",
			zap.String("key", d.key),
			zap.Int("bytes", len(value)),
		)
	} else {
		d.logger.Info("etcd dynconfig: no remote config found, using local only",
			zap.String("key", d.key),
		)
	}

	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	go d.watch(ctx, resp.Header.Revision+1)
	return nil
}

func (d *DynConfig) watch(ctx context.Context, startRev int64) {
	defer close(d.done)
	watchCh := d.client.Watch(ctx, d.key, clientv3.WithRev(startRev))
	for {
		select {
		case <-ctx.Done():
			return
		case wresp, ok := <-watchCh:
			if !ok {
				return
			}
			if wresp.Err() != nil {
				d.logger.Error("etcd dynconfig: watch error", zap.Error(wresp.Err()))
				return
			}
			for _, ev := range wresp.Events {
				if ev.Type == clientv3.EventTypeDelete {
					d.logger.Warn("etcd dynconfig: remote config key deleted, keeping current config")
					continue
				}
				d.handleUpdate(ev.Kv.Value)
			}
		}
	}
}

func (d *DynConfig) handleUpdate(value []byte) {
	tmpV := viper.New()
	tmpV.SetConfigType("yaml")
	if err := tmpV.ReadConfig(bytes.NewReader(value)); err != nil {
		d.logger.Error("etcd dynconfig: parse updated config failed", zap.Error(err))
		return
	}

	var tempCfg config.Config
	if err := tmpV.Unmarshal(&tempCfg); err != nil {
		d.logger.Error("etcd dynconfig: unmarshal updated config failed", zap.Error(err))
		return
	}

	d.mu.Lock()
	d.appCfg.HotConfig = tempCfg.HotConfig
	d.mu.Unlock()

	d.logger.Info("etcd dynconfig: hot-reloaded config fields")
}

func (d *DynConfig) Stop() {
	if d.cancel != nil {
		d.cancel()
		<-d.done
	}
	d.logger.Info("etcd dynconfig: stopped")
}
