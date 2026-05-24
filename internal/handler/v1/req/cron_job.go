package req

// CronJobUriRequest 定时任务路径参数
type CronJobUriRequest struct {
	Name string `uri:"name" binding:"required" example:"server_status"` // 任务名称
}

// CronJobDisableRequest 禁用定时任务查询参数
type CronJobDisableRequest struct {
	Force bool `form:"force"` // 是否立即取消正在执行的任务
}

// CronJobTriggerRequest 手动触发定时任务查询参数
type CronJobTriggerRequest struct {
	Force bool `form:"force"` // 是否强制执行（忽略正在运行的检查）
}

// CronJobListExecutionsRequest 执行历史分页查询参数
type CronJobListExecutionsRequest struct {
	Page     int `form:"page" binding:"omitempty,min=1" example:"1"`               // 页码
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100" example:"20"` // 每页数量
}
