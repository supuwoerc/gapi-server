package etcd

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/supuwoerc/gapi-server/internal/config"

	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type InstanceChangeEvent struct {
	ServiceName string
	Instances   []ServiceInstance
}

type InstanceChangeListener func(event InstanceChangeEvent)

type Discovery struct {
	client     *clientv3.Client
	cfg        *config.EtcdConfig
	logger     Logger
	prefixes   []string
	prefixData map[string]map[string][]ServiceInstance // prefix -> serviceName -> instances
	mu         sync.RWMutex
	listeners  []InstanceChangeListener
	listenerMu sync.RWMutex
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

func NewDiscovery(client *clientv3.Client, cfg *config.EtcdConfig, l Logger) *Discovery {
	prefixes := cfg.Discovery.Prefixes
	if len(prefixes) == 0 {
		p := cfg.Registry.Prefix
		if p == "" {
			p = "/gapi/services"
		}
		prefixes = []string{p}
	}
	return &Discovery{
		client:     client,
		cfg:        cfg,
		logger:     l,
		prefixes:   prefixes,
		prefixData: make(map[string]map[string][]ServiceInstance),
	}
}

func (d *Discovery) Start(ctx context.Context) error {
	for _, prefix := range d.prefixes {
		if err := d.loadPrefix(ctx, prefix); err != nil {
			return err
		}
	}

	d.logger.Info("etcd discovery: started", zap.Strings("prefixes", d.prefixes))

	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel

	for _, prefix := range d.prefixes {
		d.wg.Add(1)
		go d.watch(ctx, prefix)
	}
	return nil
}

func (d *Discovery) loadPrefix(ctx context.Context, prefix string) error {
	resp, err := d.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return errors.Wrapf(err, "etcd discovery: load prefix %s", prefix)
	}

	instances := make(map[string][]ServiceInstance)
	for _, kv := range resp.Kvs {
		var inst ServiceInstance
		if err := json.Unmarshal(kv.Value, &inst); err != nil {
			continue
		}
		instances[inst.ServiceName] = append(instances[inst.ServiceName], inst)
	}

	d.mu.Lock()
	d.prefixData[prefix] = instances
	d.mu.Unlock()

	d.logger.Info("etcd discovery: loaded prefix", zap.String("prefix", prefix), zap.Int("count", len(resp.Kvs)))
	return nil
}

func (d *Discovery) watch(ctx context.Context, prefix string) {
	defer d.wg.Done()
	watchCh := d.client.Watch(ctx, prefix, clientv3.WithPrefix())
	for {
		select {
		case <-ctx.Done():
			return
		case wresp, ok := <-watchCh:
			if !ok {
				return
			}
			if wresp.Err() != nil {
				d.logger.Error("etcd discovery: watch error", zap.String("prefix", prefix), zap.Error(wresp.Err()))
				return
			}
			d.reloadPrefix(ctx, prefix)
		}
	}
}

func (d *Discovery) reloadPrefix(ctx context.Context, prefix string) {
	resp, err := d.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		d.logger.Error("etcd discovery: reload failed", zap.String("prefix", prefix), zap.Error(err))
		return
	}

	instances := make(map[string][]ServiceInstance)
	for _, kv := range resp.Kvs {
		var inst ServiceInstance
		if err := json.Unmarshal(kv.Value, &inst); err != nil {
			continue
		}
		instances[inst.ServiceName] = append(instances[inst.ServiceName], inst)
	}

	d.mu.Lock()
	d.prefixData[prefix] = instances
	d.mu.Unlock()

	d.listenerMu.RLock()
	listeners := d.listeners
	d.listenerMu.RUnlock()

	for svcName, insts := range instances {
		event := InstanceChangeEvent{ServiceName: svcName, Instances: insts}
		for _, fn := range listeners {
			fn(event)
		}
	}

	d.logger.Debug("etcd discovery: reloaded prefix", zap.String("prefix", prefix), zap.Int("services", len(instances)))
}

func (d *Discovery) GetInstances(serviceName string) []ServiceInstance {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var result []ServiceInstance
	for _, svcMap := range d.prefixData {
		result = append(result, svcMap[serviceName]...)
	}
	return result
}

func (d *Discovery) Pick(serviceName string, balancer Balancer) (ServiceInstance, error) {
	instances := d.GetInstances(serviceName)
	return balancer.Pick(instances)
}

func (d *Discovery) GetAllServices() map[string][]ServiceInstance {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make(map[string][]ServiceInstance)
	for _, svcMap := range d.prefixData {
		for svcName, insts := range svcMap {
			result[svcName] = append(result[svcName], insts...)
		}
	}
	return result
}

func (d *Discovery) OnChange(fn InstanceChangeListener) {
	d.listenerMu.Lock()
	d.listeners = append(d.listeners, fn)
	d.listenerMu.Unlock()
}

func (d *Discovery) Stop() {
	if d.cancel == nil {
		return
	}
	d.cancel()
	d.wg.Wait()
	d.logger.Info("etcd discovery: stopped")
}

func (d *Discovery) OnStart(ctx context.Context) error { return d.Start(ctx) }
func (d *Discovery) OnReady(context.Context) error     { return nil }
func (d *Discovery) OnStop(context.Context) error      { d.Stop(); return nil }

func (d *Discovery) OnInit(ctx context.Context) error { return d.Start(ctx) }
func (d *Discovery) OnClose(context.Context) error    { d.Stop(); return nil }
