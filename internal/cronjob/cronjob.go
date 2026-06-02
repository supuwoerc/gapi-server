package cronjob

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type ExecutionMode int

const (
	ModeSkipIfRunning ExecutionMode = iota + 1
	ModeDelayIfRunning
	ModeAllowConcurrent
)

const (
	TriggerByScheduler = "scheduler"
	TriggerByManual    = "manual"
)

const (
	StatusRunning   = "running"
	StatusSuccess   = "success"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
	StatusPanic     = "panic"
)

var ErrJobRunning = errors.New("job is currently running")

// Logger 日志接口，解耦对具体 logger 实现的依赖。
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

type SystemJob interface {
	Name() string
	Interval() string
	ExecutionMode() ExecutionMode
	Handle(ctx context.Context) error
}

// LockResult 表示一次加锁结果，调用 Unlock 释放。
type LockResult interface {
	Unlock(ctx context.Context) error
}

// DistLocker 分布式锁抽象，用于多实例 job 执行前抢锁协调。
type DistLocker interface {
	TryLock(ctx context.Context, key string) (LockResult, error)
}
