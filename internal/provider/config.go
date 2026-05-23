package provider

import (
	"gapi-server/internal/config"

	"github.com/google/wire"
)

var ConfigSet = wire.NewSet(
	config.NewViper,
	config.NewConfig,
	ProvideLogConfig,
	ProvideDBConfig,
	ProvideServerConfig,
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
