package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/supuwoerc/gapi-server/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/lumberjack.v2"
)

type ContextKey string

const TraceIDKey ContextKey = "trace_id"

type Logger struct {
	*zap.Logger
}

func NewLogger(cfg *config.LogConfig) *Logger {
	level := parseLevel(cfg.Level)
	cores := []zapcore.Core{
		zapcore.NewCore(getJSONEncoder(), getFileSyncer(cfg), level),
	}
	if cfg.Stdout {
		cores = append(cores,
			zapcore.NewCore(getConsoleEncoder(), zapcore.AddSync(os.Stdout), level),
		)
	}
	core := zapcore.NewTee(cores...)
	z := zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.Fields(zap.Int("pid", os.Getpid())),
	)
	return &Logger{Logger: z}
}

func (l *Logger) Ctx(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return l.Logger
	}
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		return l.Logger.With(zap.String(string(TraceIDKey), traceID))
	}
	return l.Logger
}

func getJSONEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Local().Format(time.DateTime))
	}
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeCaller = zapcore.FullCallerEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

func getConsoleEncoder() zapcore.Encoder {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Local().Format(time.DateTime))
	}
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getFileSyncer(cfg *config.LogConfig) zapcore.WriteSyncer {
	dir := strings.TrimSpace(cfg.Dir)
	if dir == "" {
		projectDir, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		dir = filepath.Join(projectDir, "log")
	}
	logFileName := fmt.Sprintf("%s.log", filepath.Join(dir, time.Now().Format(time.DateOnly)))

	lj := &lumberjack.Logger{
		Filename:   logFileName,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   true,
	}
	return zapcore.AddSync(lj)
}

func parseLevel(level string) zapcore.Level {
	var l zapcore.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return zapcore.InfoLevel
	}
	return l
}
