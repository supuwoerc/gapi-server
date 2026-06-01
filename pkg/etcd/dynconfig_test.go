package etcd_test

import (
	"context"
	"testing"
	"time"

	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/pkg/etcd"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

const testKey = "/gapi/test/dynconfig/app.yaml"

type DynConfigSuite struct {
	suite.Suite
	client *clientv3.Client
	ctx    context.Context
}

func TestDynConfigSuite(t *testing.T) {
	suite.Run(t, new(DynConfigSuite))
}

func (s *DynConfigSuite) SetupSuite() {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		s.T().Skipf("etcd not available: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = client.Status(ctx, "127.0.0.1:2379")
	if err != nil {
		s.T().Skipf("etcd not reachable: %v", err)
	}
	s.client = client
	s.ctx = context.Background()
}

func (s *DynConfigSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *DynConfigSuite) TearDownTest() {
	_, _ = s.client.Delete(s.ctx, testKey)
}

func (s *DynConfigSuite) newLogger() *logger.Logger {
	l, _ := zap.NewDevelopment()
	return &logger.Logger{Logger: l}
}

func (s *DynConfigSuite) newDeps() (*config.EtcdConfig, *viper.Viper, *config.Config) {
	etcdCfg := &config.EtcdConfig{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5,
		DynConfig: config.DynConfigOptions{
			Enabled: true,
			Key:     testKey,
		},
	}
	v := viper.New()
	v.SetConfigType("yaml")
	v.Set("rate_limit.pattern", "100-M")
	v.Set("rate_limit.prefix", "gapi_limiter")
	v.Set("cors.origin_prefixes", []string{"http://localhost"})

	appCfg := &config.Config{}
	_ = v.Unmarshal(appCfg)
	return etcdCfg, v, appCfg
}

func (s *DynConfigSuite) TestStartWithRemoteConfig() {
	etcdCfg, v, appCfg := s.newDeps()

	remoteYAML := `
rate_limit:
  pattern: "500-M"
  prefix: "remote_limiter"
cors:
  origin_prefixes:
    - "https://example.com"
database:
  password: "secret_from_etcd"
`
	_, err := s.client.Put(s.ctx, testKey, remoteYAML)
	s.Require().NoError(err)

	dc := etcd.NewDynConfig(s.client, etcdCfg, v, appCfg, s.newLogger())
	err = dc.Start(s.ctx)
	s.Require().NoError(err)
	defer dc.Stop()

	s.Equal("500-M", appCfg.RateLimit.Pattern)
	s.Equal("remote_limiter", appCfg.RateLimit.Prefix)
	s.Contains(appCfg.Cors.OriginPrefixes, "https://example.com")
	s.Equal("secret_from_etcd", appCfg.Database.Password)
}

func (s *DynConfigSuite) TestStartWithoutRemoteConfig() {
	etcdCfg, v, appCfg := s.newDeps()

	dc := etcd.NewDynConfig(s.client, etcdCfg, v, appCfg, s.newLogger())
	err := dc.Start(s.ctx)
	s.Require().NoError(err)
	defer dc.Stop()

	s.Equal("100-M", appCfg.RateLimit.Pattern)
}

func (s *DynConfigSuite) TestHotReloadOnlyUpdatesAllowedFields() {
	etcdCfg, v, appCfg := s.newDeps()

	initialYAML := `
rate_limit:
  pattern: "100-M"
database:
  password: "initial_pw"
`
	_, err := s.client.Put(s.ctx, testKey, initialYAML)
	s.Require().NoError(err)

	dc := etcd.NewDynConfig(s.client, etcdCfg, v, appCfg, s.newLogger())
	err = dc.Start(s.ctx)
	s.Require().NoError(err)
	defer dc.Stop()

	s.Equal("initial_pw", appCfg.Database.Password)

	updatedYAML := `
rate_limit:
  pattern: "999-M"
  prefix: "new_prefix"
database:
  password: "changed_pw_should_not_apply"
cors:
  origin_prefixes:
    - "https://new.com"
`
	_, err = s.client.Put(s.ctx, testKey, updatedYAML)
	s.Require().NoError(err)

	time.Sleep(1 * time.Second)

	s.Equal("999-M", appCfg.RateLimit.Pattern)
	s.Equal("new_prefix", appCfg.RateLimit.Prefix)
	s.Contains(appCfg.Cors.OriginPrefixes, "https://new.com")
	s.Equal("initial_pw", appCfg.Database.Password)
}

func (s *DynConfigSuite) TestDisabled() {
	etcdCfg, v, appCfg := s.newDeps()
	etcdCfg.DynConfig.Enabled = false

	dc := etcd.NewDynConfig(s.client, etcdCfg, v, appCfg, s.newLogger())
	err := dc.Start(s.ctx)
	s.Require().NoError(err)
	dc.Stop()
}
