package provider

import (
	"github.com/supuwoerc/gapi-server/internal/config"

	"github.com/google/wire"
)

var ConfigSet = wire.NewSet(
	config.NewViper,
	config.NewConfig,
	ProvideLogConfig,
	ProvideDBConfig,
	ProvideServerConfig,
	ProvideRedisConfig,
	ProvideCorsConfig,
	ProvideRateLimitConfig,
	ProvideLocaleConfig,
)

func ProvideLogConfig(cfg *config.Config) *config.LogConfig {
	return &cfg.Log
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
