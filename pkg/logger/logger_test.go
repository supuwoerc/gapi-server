package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

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
