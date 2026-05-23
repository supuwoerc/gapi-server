package database

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type Logger struct {
	zap           *zap.Logger
	level         logger.LogLevel
	slowThreshold time.Duration
}

func NewGormLogger(zapLogger *zap.Logger, level logger.LogLevel, slowThreshold time.Duration) *Logger {
	return &Logger{
		zap:           zapLogger,
		level:         level,
		slowThreshold: slowThreshold,
	}
}

func (l *Logger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.level = level
	return &newLogger
}

func (l *Logger) Info(_ context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Info {
		l.zap.Sugar().Infof(msg, data...)
	}
}

func (l *Logger) Warn(_ context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Warn {
		l.zap.Sugar().Warnf(msg, data...)
	}
}

func (l *Logger) Error(_ context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Error {
		l.zap.Sugar().Errorf(msg, data...)
	}
}

func (l *Logger) Trace(_ context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.level <= logger.Silent {
		return
	}
	elapsed := time.Since(begin)
	cost := fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6)
	fileWithLine := utils.FileWithLineNum()

	switch {
	case err != nil && l.level >= logger.Error && !errors.Is(err, gorm.ErrRecordNotFound):
		sqlStr, affected := fc()
		l.zap.Error("gorm",
			zap.String("pos", fileWithLine),
			zap.Error(err),
			zap.String("cost", cost),
			zap.Int64("rows", affected),
			zap.String("sql", cleanSQL(sqlStr)),
		)
	case elapsed > l.slowThreshold && l.slowThreshold != 0 && l.level >= logger.Warn:
		sqlStr, affected := fc()
		l.zap.Warn("gorm slow",
			zap.String("pos", fileWithLine),
			zap.String("cost", cost),
			zap.Int64("rows", affected),
			zap.String("sql", cleanSQL(sqlStr)),
		)
	case l.level == logger.Info:
		sqlStr, affected := fc()
		l.zap.Debug("gorm",
			zap.String("pos", fileWithLine),
			zap.String("cost", cost),
			zap.Int64("rows", affected),
			zap.String("sql", cleanSQL(sqlStr)),
		)
	}
}

var multiSpaceRegexp = regexp.MustCompile(`\s+`)

func cleanSQL(sql string) string {
	if sql == "" {
		return sql
	}
	sql = strings.ReplaceAll(sql, "\n", " ")
	sql = strings.ReplaceAll(sql, "\t", " ")
	sql = multiSpaceRegexp.ReplaceAllString(sql, " ")
	return strings.TrimSpace(sql)
}
