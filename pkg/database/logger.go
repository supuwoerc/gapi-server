package database

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gapi-server/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type Logger struct {
	logger        *logger.Logger
	level         gormlogger.LogLevel
	slowThreshold time.Duration
}

func NewGormLogger(l *logger.Logger, level gormlogger.LogLevel, slowThreshold time.Duration) *Logger {
	return &Logger{
		logger:        l,
		level:         level,
		slowThreshold: slowThreshold,
	}
}

func (l *Logger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.level = level
	return &newLogger
}

func (l *Logger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= gormlogger.Info {
		l.logger.Ctx(ctx).Info(fmt.Sprintf(msg, data...))
	}
}

func (l *Logger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= gormlogger.Warn {
		l.logger.Ctx(ctx).Warn(fmt.Sprintf(msg, data...))
	}
}

func (l *Logger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= gormlogger.Error {
		l.logger.Ctx(ctx).Error(fmt.Sprintf(msg, data...))
	}
}

func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.level <= gormlogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	cost := fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6)
	fileWithLine := utils.FileWithLineNum()
	log := l.logger.Ctx(ctx)

	switch {
	case err != nil && l.level >= gormlogger.Error && !errors.Is(err, gorm.ErrRecordNotFound):
		sqlStr, affected := fc()
		log.Error("gorm",
			zap.String("pos", fileWithLine),
			zap.Error(err),
			zap.String("cost", cost),
			zap.Int64("rows", affected),
			zap.String("sql", cleanSQL(sqlStr)),
		)
	case elapsed > l.slowThreshold && l.slowThreshold != 0 && l.level >= gormlogger.Warn:
		sqlStr, affected := fc()
		log.Warn("gorm slow",
			zap.String("pos", fileWithLine),
			zap.String("cost", cost),
			zap.Int64("rows", affected),
			zap.String("sql", cleanSQL(sqlStr)),
		)
	case l.level == gormlogger.Info:
		sqlStr, affected := fc()
		log.Info("gorm",
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
