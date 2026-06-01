package config

import (
	"bytes"
	"context"
	"os"
	"time"

	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func NewViper() *viper.Viper {
	v := viper.New()
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.SetConfigName("default")
	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}
	env := DetermineEnvironment()
	v.SetConfigName(env)
	if err := v.MergeInConfig(); err != nil {
		panic(err)
	}
	return v
}

func NewBootstrapConfig(v *viper.Viper) *BootstrapConfig {
	var cfg BootstrapConfig
	if err := v.Unmarshal(&cfg); err != nil {
		panic(err)
	}
	return &cfg
}

func NewConfig(v *viper.Viper, client *clientv3.Client, bootstrap *BootstrapConfig) *Config {
	if bootstrap.Etcd.DynConfig.Enabled {
		mergeRemoteConfig(v, client, &bootstrap.Etcd)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		panic(err)
	}
	cfg.Env = DetermineEnvironment()
	return &cfg
}

func mergeRemoteConfig(v *viper.Viper, client *clientv3.Client, etcdCfg *EtcdConfig) {
	key := etcdCfg.DynConfig.Key
	if key == "" {
		return
	}
	dialTimeout := etcdCfg.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = 5
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(dialTimeout)*time.Second)
	defer cancel()
	resp, err := client.Get(ctx, key)
	if err != nil {
		panic("failed to get remote config from etcd: " + err.Error())
	}
	if len(resp.Kvs) == 0 {
		return
	}
	v.SetConfigType("yaml")
	if err := v.MergeConfig(bytes.NewReader(resp.Kvs[0].Value)); err != nil {
		panic("failed to merge remote config: " + err.Error())
	}
}

func DetermineEnvironment() string {
	env := os.Getenv("APP_ENV")
	switch env {
	case "prod", "test":
		return env
	default:
		return "dev"
	}
}
