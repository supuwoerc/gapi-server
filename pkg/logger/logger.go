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
	writeSyncer := getLogWriter(cfg)
	level := parseLevel(cfg.Level)
	encoder := getEncoder()

	core := zapcore.NewCore(encoder, writeSyncer, level)
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

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Local().Format(time.DateTime))
	}
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

func getLogWriter(cfg *config.LogConfig) zapcore.WriteSyncer {
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

	ws := []zapcore.WriteSyncer{zapcore.AddSync(lj)}
	if cfg.Stdout {
		ws = append(ws, zapcore.AddSync(os.Stdout))
	}
	return zapcore.NewMultiWriteSyncer(ws...)
}

func parseLevel(level string) zapcore.Level {
	var l zapcore.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return zapcore.InfoLevel
	}
	return l
}
