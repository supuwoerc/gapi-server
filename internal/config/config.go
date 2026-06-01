package config

import (
	"bytes"
	"context"
	"time"

	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// BootstrapConfig holds minimal configuration needed to bootstrap infrastructure (etcd, logger).
type BootstrapConfig struct {
	Etcd EtcdConfig `mapstructure:"etcd"`
	Log  LogConfig  `mapstructure:"log"`
}

// EtcdConfig holds etcd client connection settings.
type EtcdConfig struct {
	Endpoints   []string         `mapstructure:"endpoints"`    // etcd 节点地址列表
	Username    string           `mapstructure:"username"`     // 用户名
	Password    string           `mapstructure:"password"`     // 密码
	DialTimeout int              `mapstructure:"dial_timeout"` // 连接超时 (秒)
	DynConfig   DynConfigOptions `mapstructure:"dyn_config"`   // 动态配置中心
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level      string `mapstructure:"level"`       // 日志级别 (debug/info/warn/error)
	Dir        string `mapstructure:"dir"`         // 日志文件存放目录
	Stdout     bool   `mapstructure:"stdout"`      // 是否同时输出到控制台
	MaxSize    int    `mapstructure:"max_size"`    // 单个日志文件最大大小 (MB)
	MaxBackups int    `mapstructure:"max_backups"` // 保留的旧日志文件最大数量
	MaxAge     int    `mapstructure:"max_age"`     // 保留的旧日志文件最大天数
}

// Config holds all configuration sections.
type Config struct {
	HotConfig `mapstructure:",squash"`
	Server    ServerConfig   `mapstructure:"server"`   // HTTP 服务配置
	Database  DatabaseConfig `mapstructure:"database"` // 数据库配置
	Log       LogConfig      `mapstructure:"log"`      // 日志配置
	Redis     RedisConfig    `mapstructure:"redis"`    // Redis 配置
	Locale    LocaleConfig   `mapstructure:"locale"`   // 国际化配置
	Cron      CronConfig     `mapstructure:"cron"`     // 定时任务配置
	Etcd      EtcdConfig     `mapstructure:"etcd"`     // Etcd 配置
	Env       string         `mapstructure:"-"`        // 运行环境 (dev/prod/test)
}

// HotConfig holds configuration sections that can be hot-reloaded at runtime.
type HotConfig struct {
	Cors      CorsConfig      `mapstructure:"cors"`       // 跨域配置
	RateLimit RateLimitConfig `mapstructure:"rate_limit"` // 限流配置
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host               string `mapstructure:"host"`                 // 监听地址
	Port               int    `mapstructure:"port"`                 // 监听端口
	MaxMultipartMemory int64  `mapstructure:"max_multipart_memory"` // 文件上传内存限制 (MB)
}

// DatabaseConfig holds MySQL connection and pool settings.
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`              // 数据库主机地址
	Port            int    `mapstructure:"port"`              // 数据库端口
	User            string `mapstructure:"user"`              // 数据库用户名
	Password        string `mapstructure:"password"`          // 数据库密码
	DBName          string `mapstructure:"dbname"`            // 数据库名称
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`    // 连接池最大空闲连接数
	MaxOpenConns    int    `mapstructure:"max_open_conns"`    // 连接池最大打开连接数
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // 连接最大存活时间 (单位: 秒)
	SlowThreshold   int    `mapstructure:"slow_threshold"`    // 慢查询阈值 (单位: 毫秒)
	LogLevel        int    `mapstructure:"log_level"`         // GORM 日志级别 (1=Silent 2=Error 3=Warn 4=Info)
}

// CorsConfig holds CORS settings.
type CorsConfig struct {
	OriginPrefixes []string `mapstructure:"origin_prefixes"` // 允许的 Origin 前缀列表
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	Pattern string `mapstructure:"pattern"` // 限流模式 (如 "100-M" = 100次/分钟)
	Prefix  string `mapstructure:"prefix"`  // Redis key 前缀
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr               string `mapstructure:"addr"`                // Redis 地址
	Password           string `mapstructure:"password"`            // Redis 密码
	DB                 int    `mapstructure:"db"`                  // Redis 数据库编号
	LogLevel           int    `mapstructure:"log_level"`           // 日志级别 (1=Silent 2=Error 3=Warn 4=Info)
	MaintNotifications string `mapstructure:"maint_notifications"` // maint_notifications 模式 (disabled/enabled/auto)
}

// LocaleConfig holds i18n settings.
type LocaleConfig struct {
	DefaultLang string `mapstructure:"default_lang"` // 默认语言 (cn/en)
	LocaleKey   string `mapstructure:"locale_key"`   // 请求 header key
}

// CronConfig holds cron job scheduler settings.
type CronConfig struct {
	Enabled         bool `mapstructure:"enabled"`          // 是否启用定时任务
	ShutdownTimeout int  `mapstructure:"shutdown_timeout"` // 关闭时等待运行中任务的超时时间 (秒)
}

// DynConfigOptions holds dynamic configuration center settings.
type DynConfigOptions struct {
	Enabled bool   `mapstructure:"enabled"` // 是否启用远程配置
	Key     string `mapstructure:"key"`     // etcd 中存储完整 YAML 的 key
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
