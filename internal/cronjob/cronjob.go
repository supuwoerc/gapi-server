package cronjob

import "context"

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

type SystemJob interface {
	Name() string
	Interval() string
	ExecutionMode() ExecutionMode
	Handle(ctx context.Context) error
}
