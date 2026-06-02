package cronjob

import (
	"context"

	"github.com/pkg/errors"
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
