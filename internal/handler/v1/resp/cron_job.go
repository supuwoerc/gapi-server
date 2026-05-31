package resp

import (
	"time"

	"github.com/supuwoerc/gapi-server/internal/dal/model"
)

type CronJobItem struct {
	ID          uint64     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Interval    string     `json:"interval"`
	Enabled     bool       `json:"enabled"`
	LastRunAt   *time.Time `json:"last_run_at"`
	LastStatus  string     `json:"last_status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func NewCronJobItem(m *model.CronJob) *CronJobItem {
	return &CronJobItem{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Interval:    m.Interval,
		Enabled:     m.Enabled,
		LastRunAt:   m.LastRunAt,
		LastStatus:  m.LastStatus,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func NewCronJobItems(ms []*model.CronJob) []*CronJobItem {
	items := make([]*CronJobItem, 0, len(ms))
	for _, m := range ms {
		items = append(items, NewCronJobItem(m))
	}
	return items
}

type CronJobExecutionItem struct {
	ID          uint64     `json:"id"`
	JobName     string     `json:"job_name"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at"`
	Duration    *int64     `json:"duration"`
	Error       string     `json:"error"`
	TriggeredBy string     `json:"triggered_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

func NewCronJobExecutionItem(m *model.CronJobExecution) *CronJobExecutionItem {
	return &CronJobExecutionItem{
		ID:          m.ID,
		JobName:     m.JobName,
		Status:      m.Status,
		StartedAt:   m.StartedAt,
		EndedAt:     m.EndedAt,
		Duration:    m.Duration,
		Error:       m.Error,
		TriggeredBy: m.TriggeredBy,
		CreatedAt:   m.CreatedAt,
	}
}

func NewCronJobExecutionItems(ms []*model.CronJobExecution) []*CronJobExecutionItem {
	items := make([]*CronJobExecutionItem, 0, len(ms))
	for _, m := range ms {
		items = append(items, NewCronJobExecutionItem(m))
	}
	return items
}

// CronJobListResponse 定时任务列表响应
type CronJobListResponse struct {
	Jobs []*CronJobItem `json:"jobs"`
}

// CronJobDetailResponse 定时任务详情响应
type CronJobDetailResponse struct {
	*CronJobItem
}

// CronJobListExecutionsResponse 执行历史分页响应
type CronJobListExecutionsResponse struct {
	Total      int64                   `json:"total"`
	Executions []*CronJobExecutionItem `json:"executions"`
}
