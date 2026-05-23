package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestCtx(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	l := &Logger{Logger: zap.New(core)}

	tests := []struct {
		name        string
		ctx         context.Context
		hasTraceID  bool
		wantTraceID string
	}{
		{
			name:       "nil context returns base logger",
			ctx:        nil,
			hasTraceID: false,
		},
		{
			name:       "context without trace_id returns base logger",
			ctx:        context.Background(),
			hasTraceID: false,
		},
		{
			name:        "context with trace_id injects field",
			ctx:         context.WithValue(context.Background(), TraceIDKey, "abc-123"),
			hasTraceID:  true,
			wantTraceID: "abc-123",
		},
		{
			name:       "context with empty trace_id returns base logger",
			ctx:        context.WithValue(context.Background(), TraceIDKey, ""),
			hasTraceID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorded.TakeAll()
			l.Ctx(tt.ctx).Info("test")

			entries := recorded.All()
			assert.Len(t, entries, 1)

			fields := entries[0].ContextMap()
			if tt.hasTraceID {
				assert.Equal(t, tt.wantTraceID, fields[string(TraceIDKey)])
			} else {
				_, exists := fields[string(TraceIDKey)]
				assert.False(t, exists)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  zapcore.Level
	}{
		{name: "debug", input: "debug", want: zapcore.DebugLevel},
		{name: "info", input: "info", want: zapcore.InfoLevel},
		{name: "warn", input: "warn", want: zapcore.WarnLevel},
		{name: "error", input: "error", want: zapcore.ErrorLevel},
		{name: "dpanic", input: "dpanic", want: zapcore.DPanicLevel},
		{name: "panic", input: "panic", want: zapcore.PanicLevel},
		{name: "fatal", input: "fatal", want: zapcore.FatalLevel},
		{name: "empty defaults to info", input: "", want: zapcore.InfoLevel},
		{name: "invalid defaults to info", input: "unknown", want: zapcore.InfoLevel},
		{name: "case insensitive DEBUG", input: "DEBUG", want: zapcore.DebugLevel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseLevel(tt.input))
		})
	}
}
