package model

import "time"

type CronJob struct {
	BaseModel
	Name        string     `json:"name" gorm:"type:varchar(128);uniqueIndex;not null;comment:任务唯一标识"`
	Description string     `json:"description" gorm:"type:varchar(512);default:'';comment:任务描述"`
	Interval    string     `json:"interval" gorm:"type:varchar(64);not null;comment:cron表达式"`
	Enabled     bool       `json:"enabled" gorm:"default:true;comment:是否启用"`
	LastRunAt   *time.Time `json:"last_run_at" gorm:"comment:最近一次执行时间"`
	LastStatus  string     `json:"last_status" gorm:"type:varchar(16);default:'';comment:最近一次执行状态"`
}

func (CronJob) TableName() string { return "sys_cron_job" }
