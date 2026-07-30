package database

import (
	"fmt"
	"time"

	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

const tablePrefix = "sys_"

func NewConnection(cfg *config.DatabaseConfig, l *logger.Logger) (*gorm.DB, error) {
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	timeZone := cfg.TimeZone
	if timeZone == "" {
		timeZone = "Asia/Shanghai"
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.DBName,
		sslMode,
		timeZone,
	)

	logLevel := gormlogger.LogLevel(cfg.LogLevel)
	if logLevel == 0 {
		logLevel = gormlogger.Warn
	}
	slowThreshold := time.Duration(cfg.SlowThreshold) * time.Millisecond
	if slowThreshold == 0 {
		slowThreshold = 200 * time.Millisecond
	}
	gormLogger := NewGormLogger(l, logLevel, slowThreshold)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   tablePrefix,
			SingularTable: true,
		},
		Logger:         gormLogger,
		TranslateError: true,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get underlying sql.DB")
	}

	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 10
	}
	maxOpenConns := cfg.MaxOpenConns
	if maxOpenConns == 0 {
		maxOpenConns = 100
	}
	connMaxLifetime := time.Duration(cfg.ConnMaxLifetime) * time.Second
	if connMaxLifetime == 0 {
		connMaxLifetime = time.Hour
	}

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	return db, nil
}
