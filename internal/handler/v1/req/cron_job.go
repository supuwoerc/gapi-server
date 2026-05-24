package req

// CronJobUriRequest 定时任务路径参数
type CronJobUriRequest struct {
	Name string `uri:"name" binding:"required" example:"server_status"` // 任务名称
}

// CronJobSetEnabledRequest 启用/禁用定时任务请求
type CronJobSetEnabledRequest struct {
	Enabled *bool `json:"enabled" binding:"required" example:"true"` // 是否启用
}

// CronJobListExecutionsRequest 执行历史分页查询参数
type CronJobListExecutionsRequest struct {
	Page     int `form:"page" binding:"omitempty,min=1" example:"1"`               // 页码
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100" example:"20"` // 每页数量
}
