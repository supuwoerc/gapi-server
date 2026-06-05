package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/pkg/netutil"

	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type ServiceInstance struct {
	ServiceName string            `json:"service_name"`
	InstanceID  string            `json:"instance_id"`
	Addr        string            `json:"addr"`
	Weight      int               `json:"weight"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type Registry struct {
	client  *clientv3.Client
	cfg     *config.EtcdConfig
	srvCfg  *config.ServerConfig
	logger  Logger
	leaseID clientv3.LeaseID
	key     string
	cancel  context.CancelFunc
	done    chan struct{}
}

func NewRegistry(client *clientv3.Client, cfg *config.EtcdConfig, srvCfg *config.ServerConfig, l Logger) *Registry {
	return &Registry{
		client: client,
		cfg:    cfg,
		srvCfg: srvCfg,
		logger: l,
		done:   make(chan struct{}),
	}
}

func (r *Registry) Register(ctx context.Context) error {
	if !r.cfg.Registry.Enabled {
		r.logger.Info("etcd registry: disabled by config")
		return nil
	}

	outboundIP, err := netutil.OutboundIP()
	if err != nil {
		return errors.Wrap(err, "etcd registry: get outbound IP")
	}

	ttl := int64(r.cfg.Registry.TTL)
	if ttl <= 0 {
		ttl = 10
	}

	lease, err := r.client.Grant(ctx, ttl)
	if err != nil {
		return errors.Wrap(err, "etcd registry: grant lease")
	}
	r.leaseID = lease.ID

	weight := r.cfg.Registry.Weight
	if weight <= 0 {
		weight = 100
	}

	instanceID := r.buildInstanceID()
	addr := fmt.Sprintf("%s:%d", outboundIP.String(), r.srvCfg.Port)

	instance := ServiceInstance{
		ServiceName: r.cfg.Registry.ServiceName,
		InstanceID:  instanceID,
		Addr:        addr,
		Weight:      weight,
	}
	val, _ := json.Marshal(instance)

	r.key = r.buildKey(instanceID)
	_, err = r.client.Put(ctx, r.key, string(val), clientv3.WithLease(lease.ID))
	if err != nil {
		return errors.Wrap(err, "etcd registry: put instance")
	}

	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	go r.keepAlive(ctx)

	r.logger.Info("etcd registry: registered",
		zap.String("key", r.key),
		zap.String("addr", addr),
		zap.Int("weight", weight),
	)
	return nil
}

func (r *Registry) Deregister() {
	if r.cancel == nil {
		return
	}
	r.cancel()
	<-r.done

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if r.key != "" {
		_, _ = r.client.Delete(ctx, r.key)
	}
	if r.leaseID != 0 {
		_, _ = r.client.Revoke(ctx, r.leaseID)
	}
	r.logger.Info("etcd registry: deregistered", zap.String("key", r.key))
}

func (r *Registry) buildInstanceID() string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("%s-%d-%d", hostname, r.srvCfg.Port, os.Getpid())
}

func (r *Registry) buildKey(instanceID string) string {
	prefix := r.cfg.Registry.Prefix
	if prefix == "" {
		prefix = "/gapi/services"
	}
	return fmt.Sprintf("%s/%s/%s", prefix, r.cfg.Registry.ServiceName, instanceID)
}

func (r *Registry) keepAlive(ctx context.Context) {
	defer close(r.done)
	ch, err := r.client.KeepAlive(ctx, r.leaseID)
	if err != nil {
		r.logger.Error("etcd registry: keepalive failed", zap.Error(err))
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case resp, ok := <-ch:
			if !ok {
				r.logger.Warn("etcd registry: keepalive channel closed")
				return
			}
			if resp != nil {
				r.logger.Debug("etcd registry: keepalive", zap.Int64("ttl", resp.TTL))
			}
		}
	}
}
