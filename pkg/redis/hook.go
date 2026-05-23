package redis

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"gapi-server/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type LogLevel int

const (
	Silent LogLevel = iota + 1
	Error
	Warn
	Info
)

type Hook struct {
	logger *logger.Logger
	level  LogLevel
}

func NewHook(l *logger.Logger, level LogLevel) *Hook {
	return &Hook{
		logger: l,
		level:  level,
	}
}

func (h *Hook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if h.level >= Info {
			h.logger.Ctx(ctx).Info("dialing to Redis",
				zap.String("network", network),
				zap.String("addr", addr),
			)
		}
		conn, err := next(ctx, network, addr)
		if err != nil && h.level >= Error {
			h.logger.Ctx(ctx).Error("dialing error", zap.Error(err))
		} else if err == nil && h.level >= Info {
			h.logger.Ctx(ctx).Info("connected to Redis",
				zap.String("network", network),
				zap.String("addr", addr),
			)
		}
		return conn, err
	}
}

func (h *Hook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if h.level >= Info {
			h.logger.Ctx(ctx).Info("executing command",
				zap.String("command", cmd.Name()),
				zap.String("args", buildArgs(cmd)),
			)
		}
		err := next(ctx, cmd)
		if err != nil && h.level >= Error {
			if errors.Is(err, redis.Nil) {
				h.logger.Ctx(ctx).Info("command returned nil", zap.String("command", cmd.Name()))
			} else {
				h.logger.Ctx(ctx).Error("error executing command",
					zap.String("command", cmd.Name()),
					zap.Error(err),
				)
			}
		} else if err == nil && h.level >= Info {
			h.logger.Ctx(ctx).Info("command executed", zap.String("command", cmd.Name()))
		}
		return err
	}
}

func (h *Hook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		if h.level >= Info {
			names := make([]string, len(cmds))
			for i, cmd := range cmds {
				names[i] = cmd.Name()
			}
			h.logger.Ctx(ctx).Info("executing pipeline",
				zap.String("commands", strings.Join(names, ", ")),
			)
		}
		err := next(ctx, cmds)
		if err != nil && h.level >= Error {
			if errors.Is(err, redis.Nil) {
				h.logger.Ctx(ctx).Info("pipeline returned nil")
			} else {
				h.logger.Ctx(ctx).Error("error executing pipeline", zap.Error(err))
			}
		} else if err == nil && h.level >= Info {
			h.logger.Ctx(ctx).Info("pipeline executed")
		}
		return err
	}
}

func buildArgs(cmd redis.Cmder) string {
	args := cmd.Args()
	if len(args) <= 1 {
		return ""
	}
	var b strings.Builder
	for i := 1; i < len(args); i++ {
		if i > 1 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "%v", args[i])
	}
	return b.String()
}
