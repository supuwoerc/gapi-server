package model

import "time"

type CronJobExecution struct {
	BaseModel
	JobName     string     `json:"job_name" gorm:"type:varchar(128);index;not null;comment:任务名称"`
	Status      string     `json:"status" gorm:"type:varchar(16);not null;comment:执行状态"`
	StartedAt   time.Time  `json:"started_at" gorm:"not null;comment:开始时间"`
	EndedAt     *time.Time `json:"ended_at" gorm:"comment:结束时间"`
	Duration    *int64     `json:"duration" gorm:"comment:耗时(毫秒)"`
	Error       string     `json:"error" gorm:"type:text;comment:错误信息"`
	TriggeredBy string     `json:"triggered_by" gorm:"type:varchar(32);default:'scheduler';comment:触发方式"`
}

func (CronJobExecution) TableName() string { return "sys_cron_job_execution" }
