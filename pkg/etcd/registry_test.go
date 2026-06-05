package etcd_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/pkg/etcd"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func setupTestClient(t *testing.T) *clientv3.Client {
	t.Helper()
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Skipf("etcd not available: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err = client.Status(ctx, "127.0.0.1:2379"); err != nil {
		t.Skipf("etcd not reachable: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func setupTestLogger() *logger.Logger {
	cfg := &config.LogConfig{
		Level:  "debug",
		Stdout: true,
	}
	return logger.NewLogger(cfg)
}

func setupRegistryConfig() (*config.EtcdConfig, *config.ServerConfig) {
	etcdCfg := &config.EtcdConfig{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5,
	}
	etcdCfg.Registry.Enabled = true
	etcdCfg.Registry.ServiceName = "test-service"
	etcdCfg.Registry.TTL = 5
	etcdCfg.Registry.Prefix = "/gapi/test/services"
	etcdCfg.Registry.Weight = 100

	srvCfg := &config.ServerConfig{
		Host: "0.0.0.0",
		Port: 9090,
	}
	return etcdCfg, srvCfg
}

func TestRegistry_RegisterAndDeregister(t *testing.T) {
	client := setupTestClient(t)
	etcdCfg, srvCfg := setupRegistryConfig()

	ctx := context.Background()
	_, _ = client.Delete(ctx, etcdCfg.Registry.Prefix, clientv3.WithPrefix())
	t.Cleanup(func() { _, _ = client.Delete(ctx, etcdCfg.Registry.Prefix, clientv3.WithPrefix()) })

	reg := etcd.NewRegistry(client, etcdCfg, srvCfg, setupTestLogger())
	err := reg.Register(ctx)
	require.NoError(t, err)

	resp, err := client.Get(ctx, etcdCfg.Registry.Prefix+"/test-service", clientv3.WithPrefix())
	require.NoError(t, err)
	assert.Equal(t, 1, int(resp.Count))

	var inst etcd.ServiceInstance
	err = json.Unmarshal(resp.Kvs[0].Value, &inst)
	require.NoError(t, err)
	assert.Equal(t, "test-service", inst.ServiceName)
	assert.Equal(t, 100, inst.Weight)
	assert.Contains(t, inst.Addr, ":9090")
	assert.NotContains(t, inst.Addr, "0.0.0.0")

	reg.Deregister()

	resp, err = client.Get(ctx, etcdCfg.Registry.Prefix+"/test-service", clientv3.WithPrefix())
	require.NoError(t, err)
	assert.Equal(t, 0, int(resp.Count))
}

func TestRegistry_Disabled(t *testing.T) {
	client := setupTestClient(t)
	etcdCfg, srvCfg := setupRegistryConfig()
	etcdCfg.Registry.Enabled = false

	reg := etcd.NewRegistry(client, etcdCfg, srvCfg, setupTestLogger())
	err := reg.Register(context.Background())
	require.NoError(t, err)
	reg.Deregister()
}
