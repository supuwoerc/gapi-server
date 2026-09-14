package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	v := viper.New()
	v.Set("server.host", "127.0.0.1")
	v.Set("server.port", 9090)
	v.Set("database.host", "localhost")
	v.Set("database.port", 5432)
	v.Set("database.user", "admin")
	v.Set("database.password", "secret")
	v.Set("database.dbname", "testdb")
	v.Set("database.sslmode", "disable")
	v.Set("database.timezone", "Asia/Shanghai")
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
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "admin", cfg.Database.User)
	assert.Equal(t, "secret", cfg.Database.Password)
	assert.Equal(t, "testdb", cfg.Database.DBName)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, "Asia/Shanghai", cfg.Database.TimeZone)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "/tmp/logs", cfg.Log.Dir)
	assert.True(t, cfg.Log.Stdout)
	assert.Equal(t, 50, cfg.Log.MaxSize)
	assert.Equal(t, 3, cfg.Log.MaxBackups)
	assert.Equal(t, 7, cfg.Log.MaxAge)
}

// mapstructure tag 与 yaml key 的对应关系编译器管不到，写错只会静默落到零值而非报错，
// 所以直接拿仓库里那份真实的 default.yaml 反序列化一次，钉住容易写错的几段。
func TestDefaultYamlUnmarshalsIntoConfig(t *testing.T) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile("../../configs/default.yaml")
	require.NoError(t, v.ReadInConfig())

	var cfg Config
	require.NoError(t, v.Unmarshal(&cfg))

	// 远程配置：这一段只能来自本地文件，key 写错等于远程配置静默不生效
	assert.True(t, cfg.Etcd.RemoteConfig.Enabled, "etcd.remote_config.enabled 没读到")
	assert.Equal(t, "/gapi/config/app.yaml", cfg.Etcd.RemoteConfig.Key)

	// 原先内嵌在 HotConfig 里、现已平铺进 Config 的三段
	assert.Equal(t, []string{"http://localhost"}, cfg.Cors.OriginPrefixes)
	assert.Equal(t, "100-M", cfg.RateLimit.Pattern)
	assert.Equal(t, "gapi_limiter", cfg.RateLimit.Prefix)
	assert.Equal(t, []string{"welcome", "dashboard"}, cfg.Tour.ValidIDs)
}
