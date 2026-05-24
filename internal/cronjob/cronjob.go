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
