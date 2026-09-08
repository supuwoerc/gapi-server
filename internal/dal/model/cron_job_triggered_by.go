package model

import "database/sql/driver"

// TriggeredBy 定时任务的触发方式，存进 sys_cron_job_execution.triggered_by
// （VARCHAR(32)，CHECK IN ('scheduler','manual')）。
// 常量值即落库值，改拼写要同步 migrations 里的 CHECK。
type TriggeredBy string

const (
	TriggeredByScheduler TriggeredBy = "scheduler"
	TriggeredByManual    TriggeredBy = "manual"
)

func (i *TriggeredBy) Scan(src any) error          { return enumStringFromDB(i, src) }
func (i TriggeredBy) Value() (driver.Value, error) { return enumStringIntoDB(i) }
func (i TriggeredBy) Text() string                 { return enumStringText(i) }
