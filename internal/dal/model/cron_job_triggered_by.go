package model

type TriggeredBy string

const (
	TriggeredByScheduler TriggeredBy = "scheduler"
	TriggeredByManual    TriggeredBy = "manual"
)
