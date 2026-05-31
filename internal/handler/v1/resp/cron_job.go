package resp

import "github.com/supuwoerc/gapi-server/internal/dal/model"

// CronJobListResponse 定时任务列表响应
type CronJobListResponse struct {
	Jobs []*model.CronJob `json:"jobs"` // 任务列表
}

// CronJobDetailResponse 定时任务详情响应
type CronJobDetailResponse struct {
	*model.CronJob
}

// CronJobListExecutionsResponse 执行历史分页响应
type CronJobListExecutionsResponse struct {
	Total      int64                     `json:"total"`      // 总数
	Executions []*model.CronJobExecution `json:"executions"` // 执行记录列表
}
