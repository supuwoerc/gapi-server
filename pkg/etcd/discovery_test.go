package etcd_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/pkg/etcd"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func setupDiscoveryConfig() *config.EtcdConfig {
	cfg := &config.EtcdConfig{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5,
	}
	cfg.Discovery.Prefixes = []string{"/gapi/test/discovery"}
	return cfg
}

func TestDiscovery_FindInstances(t *testing.T) {
	client := setupTestClient(t)
	cfg := setupDiscoveryConfig()

	ctx := context.Background()
	prefix := cfg.Discovery.Prefixes[0]
	_, _ = client.Delete(ctx, prefix, clientv3.WithPrefix())
	t.Cleanup(func() { _, _ = client.Delete(ctx, prefix, clientv3.WithPrefix()) })

	inst1 := etcd.ServiceInstance{ServiceName: "svc-a", InstanceID: "node-1", Addr: "10.0.0.1:8080", Weight: 100}
	inst2 := etcd.ServiceInstance{ServiceName: "svc-a", InstanceID: "node-2", Addr: "10.0.0.2:8080", Weight: 200}
	val1, _ := json.Marshal(inst1)
	val2, _ := json.Marshal(inst2)
	_, _ = client.Put(ctx, prefix+"/svc-a/node-1", string(val1))
	_, _ = client.Put(ctx, prefix+"/svc-a/node-2", string(val2))

	disc := etcd.NewDiscovery(client, cfg, setupTestLogger())
	err := disc.Start(ctx)
	require.NoError(t, err)
	defer disc.Stop()

	instances := disc.GetInstances("svc-a")
	assert.Len(t, instances, 2)
}

func TestDiscovery_WatchNewInstance(t *testing.T) {
	client := setupTestClient(t)
	cfg := setupDiscoveryConfig()

	ctx := context.Background()
	prefix := cfg.Discovery.Prefixes[0]
	_, _ = client.Delete(ctx, prefix, clientv3.WithPrefix())
	t.Cleanup(func() { _, _ = client.Delete(ctx, prefix, clientv3.WithPrefix()) })

	disc := etcd.NewDiscovery(client, cfg, setupTestLogger())
	err := disc.Start(ctx)
	require.NoError(t, err)
	defer disc.Stop()

	assert.Empty(t, disc.GetInstances("svc-b"))

	inst := etcd.ServiceInstance{ServiceName: "svc-b", InstanceID: "node-3", Addr: "10.0.0.3:8080", Weight: 100}
	val, _ := json.Marshal(inst)
	_, err = client.Put(ctx, prefix+"/svc-b/node-3", string(val))
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	instances := disc.GetInstances("svc-b")
	assert.Len(t, instances, 1)
	assert.Equal(t, "10.0.0.3:8080", instances[0].Addr)
}

func TestDiscovery_WatchRemoveInstance(t *testing.T) {
	client := setupTestClient(t)
	cfg := setupDiscoveryConfig()

	ctx := context.Background()
	prefix := cfg.Discovery.Prefixes[0]
	_, _ = client.Delete(ctx, prefix, clientv3.WithPrefix())
	t.Cleanup(func() { _, _ = client.Delete(ctx, prefix, clientv3.WithPrefix()) })

	inst := etcd.ServiceInstance{ServiceName: "svc-c", InstanceID: "node-4", Addr: "10.0.0.4:8080", Weight: 100}
	val, _ := json.Marshal(inst)
	_, _ = client.Put(ctx, prefix+"/svc-c/node-4", string(val))

	disc := etcd.NewDiscovery(client, cfg, setupTestLogger())
	err := disc.Start(ctx)
	require.NoError(t, err)
	defer disc.Stop()

	assert.Len(t, disc.GetInstances("svc-c"), 1)

	_, _ = client.Delete(ctx, prefix+"/svc-c/node-4")

	time.Sleep(1 * time.Second)
	assert.Empty(t, disc.GetInstances("svc-c"))
}

func TestDiscovery_OnChangeCallback(t *testing.T) {
	client := setupTestClient(t)
	cfg := setupDiscoveryConfig()

	ctx := context.Background()
	prefix := cfg.Discovery.Prefixes[0]
	_, _ = client.Delete(ctx, prefix, clientv3.WithPrefix())
	t.Cleanup(func() { _, _ = client.Delete(ctx, prefix, clientv3.WithPrefix()) })

	disc := etcd.NewDiscovery(client, cfg, setupTestLogger())

	var received []etcd.InstanceChangeEvent
	disc.OnChange(func(event etcd.InstanceChangeEvent) {
		received = append(received, event)
	})

	err := disc.Start(ctx)
	require.NoError(t, err)
	defer disc.Stop()

	inst := etcd.ServiceInstance{ServiceName: "svc-d", InstanceID: "node-5", Addr: "10.0.0.5:8080", Weight: 100}
	val, _ := json.Marshal(inst)
	_, _ = client.Put(ctx, prefix+"/svc-d/node-5", string(val))

	time.Sleep(1 * time.Second)
	assert.NotEmpty(t, received)
}
