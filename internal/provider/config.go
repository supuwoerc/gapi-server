package provider

import (
	"github.com/supuwoerc/gapi-server/internal/config"

	"github.com/google/wire"
)

var ConfigSet = wire.NewSet(
	config.NewViper,
	config.NewBootstrapConfig,
	config.NewConfig,
	ProvideLogConfig,
	ProvideEtcdConfig,
	ProvideDBConfig,
	ProvideServerConfig,
	ProvideRedisConfig,
	ProvideCorsConfig,
	ProvideRateLimitConfig,
	ProvideLocaleConfig,
	ProvideCronConfig,
	ProvideJWTConfig,
)

func ProvideLogConfig(cfg *config.BootstrapConfig) *config.LogConfig {
	return &cfg.Log
}

func ProvideEtcdConfig(cfg *config.BootstrapConfig) *config.EtcdConfig {
	return &cfg.Etcd
}

func ProvideDBConfig(cfg *config.Config) *config.DatabaseConfig {
	return &cfg.Database
}

func ProvideServerConfig(cfg *config.Config) *config.ServerConfig {
	return &cfg.Server
}

func ProvideRedisConfig(cfg *config.Config) *config.RedisConfig {
	return &cfg.Redis
}

func ProvideCorsConfig(cfg *config.Config) *config.CorsConfig {
	return &cfg.Cors
}

func ProvideRateLimitConfig(cfg *config.Config) *config.RateLimitConfig {
	return &cfg.RateLimit
}

func ProvideLocaleConfig(cfg *config.Config) *config.LocaleConfig {
	return &cfg.Locale
}

func ProvideCronConfig(cfg *config.Config) *config.CronConfig {
	return &cfg.Cron
}

func ProvideJWTConfig(cfg *config.Config) *config.JWTConfig {
	return &cfg.JWT
}
