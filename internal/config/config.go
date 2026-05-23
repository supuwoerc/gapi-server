package config

import (
	"os"

	"github.com/spf13/viper"
)

// Config holds all configuration sections.
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`     // HTTP 服务配置
	Database  DatabaseConfig  `mapstructure:"database"`   // 数据库配置
	Log       LogConfig       `mapstructure:"log"`        // 日志配置
	Cors      CorsConfig      `mapstructure:"cors"`       // 跨域配置
	RateLimit RateLimitConfig `mapstructure:"rate_limit"` // 限流配置
	Redis     RedisConfig     `mapstructure:"redis"`      // Redis 配置
	Locale    LocaleConfig    `mapstructure:"locale"`     // 国际化配置
	Env       string          `mapstructure:"-"`          // 运行环境 (dev/prod/test)
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

// LogConfig holds logging settings.
type LogConfig struct {
	Level      string `mapstructure:"level"`       // 日志级别 (debug/info/warn/error)
	Dir        string `mapstructure:"dir"`         // 日志文件存放目录
	Stdout     bool   `mapstructure:"stdout"`      // 是否同时输出到控制台
	MaxSize    int    `mapstructure:"max_size"`    // 单个日志文件最大大小 (MB)
	MaxBackups int    `mapstructure:"max_backups"` // 保留的旧日志文件最大数量
	MaxAge     int    `mapstructure:"max_age"`     // 保留的旧日志文件最大天数
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

// NewConfig unmarshals viper config into Config struct.
func NewConfig(v *viper.Viper) *Config {
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		panic(err)
	}
	cfg.Env = os.Getenv("APP_ENV")
	if cfg.Env == "" {
		cfg.Env = "dev"
	}
	return &cfg
}
