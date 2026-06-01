package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	v := viper.New()
	v.Set("server.host", "127.0.0.1")
	v.Set("server.port", 9090)
	v.Set("database.host", "localhost")
	v.Set("database.port", 3306)
	v.Set("database.user", "admin")
	v.Set("database.password", "secret")
	v.Set("database.dbname", "testdb")
	v.Set("log.level", "debug")
	v.Set("log.dir", "/tmp/logs")
	v.Set("log.stdout", true)
	v.Set("log.max_size", 50)
	v.Set("log.max_backups", 3)
	v.Set("log.max_age", 7)

	bootstrap := &BootstrapConfig{}
	cfg := NewConfig(v, nil, bootstrap)

	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 3306, cfg.Database.Port)
	assert.Equal(t, "admin", cfg.Database.User)
	assert.Equal(t, "secret", cfg.Database.Password)
	assert.Equal(t, "testdb", cfg.Database.DBName)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "/tmp/logs", cfg.Log.Dir)
	assert.True(t, cfg.Log.Stdout)
	assert.Equal(t, 50, cfg.Log.MaxSize)
	assert.Equal(t, 3, cfg.Log.MaxBackups)
	assert.Equal(t, 7, cfg.Log.MaxAge)
}
