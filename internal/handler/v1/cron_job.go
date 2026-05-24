package v1

import (
	"strconv"

	"github.com/supuwoerc/gapi-server/internal/cronjob"
	"github.com/supuwoerc/gapi-server/internal/service"
	"github.com/supuwoerc/gapi-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type CronJobHandler struct {
	service    *service.CronJobService
	jobManager *cronjob.JobManager
}

func NewCronJobHandler(svc *service.CronJobService, jobManager *cronjob.JobManager) *CronJobHandler {
	return &CronJobHandler{service: svc, jobManager: jobManager}
}

// Register registers cron job admin routes.
// TODO: 后续认证模块完善后需要添加鉴权中间件
func (h *CronJobHandler) Register(r *gin.RouterGroup) {
	jobs := r.Group("/cron-jobs")
	{
		jobs.GET("", h.List)
		jobs.GET("/:name", h.Get)
		jobs.PUT("/:name/enabled", h.SetEnabled)
		jobs.POST("/:name/trigger", h.Trigger)
		jobs.GET("/:name/executions", h.ListExecutions)
	}
}

type setEnabledRequest struct {
	Enabled *bool `json:"enabled" binding:"required"`
}

// List
// @Summary      列出所有定时任务
// @Description  返回所有已注册的定时任务及其状态
// @Tags         定时任务
// @Produce      json
// @Success      200  {object}  response.Response
// @Router       /api/v1/cron-jobs [get]
func (h *CronJobHandler) List(c *gin.Context) {
	jobs, err := h.service.ListJobs(c.Request.Context())
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.SuccessWithData(c, jobs)
}

// Get
// @Summary      获取定时任务详情
// @Description  根据名称获取定时任务详情
// @Tags         定时任务
// @Produce      json
// @Param        name  path  string  true  "任务名称"
// @Success      200  {object}  response.Response
// @Router       /api/v1/cron-jobs/{name} [get]
func (h *CronJobHandler) Get(c *gin.Context) {
	name := c.Param("name")
	job, err := h.service.GetJob(c.Request.Context(), name)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.SuccessWithData(c, job)
}

// SetEnabled
// @Summary      启用/禁用定时任务
// @Description  修改定时任务的启用状态
// @Tags         定时任务
// @Accept       json
// @Produce      json
// @Param        name  path  string  true  "任务名称"
// @Param        body  body  setEnabledRequest  true  "启用状态"
// @Success      200  {object}  response.Response
// @Router       /api/v1/cron-jobs/{name}/enabled [put]
func (h *CronJobHandler) SetEnabled(c *gin.Context) {
	name := c.Param("name")
	var req setEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	if err := h.service.SetEnabled(c.Request.Context(), name, *req.Enabled); err != nil {
		response.FailWithError(c, err)
		return
	}
	ctx := c.Request.Context()
	if *req.Enabled {
		if err := h.jobManager.EnableJob(ctx, name); err != nil {
			response.FailWithError(c, err)
			return
		}
	} else {
		if err := h.jobManager.DisableJob(ctx, name); err != nil {
			response.FailWithError(c, err)
			return
		}
	}
	response.Success(c)
}

// Trigger
// @Summary      手动触发定时任务
// @Description  立即执行一次指定的定时任务
// @Tags         定时任务
// @Produce      json
// @Param        name  path  string  true  "任务名称"
// @Success      200  {object}  response.Response
// @Router       /api/v1/cron-jobs/{name}/trigger [post]
func (h *CronJobHandler) Trigger(c *gin.Context) {
	name := c.Param("name")
	if err := h.jobManager.TriggerManual(c.Request.Context(), name); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c)
}

// ListExecutions
// @Summary      查看定时任务执行历史
// @Description  分页查看指定任务的执行记录
// @Tags         定时任务
// @Produce      json
// @Param        name       path   string  true   "任务名称"
// @Param        page       query  int     false  "页码"
// @Param        page_size  query  int     false  "每页数量"
// @Success      200  {object}  response.Response
// @Router       /api/v1/cron-jobs/{name}/executions [get]
func (h *CronJobHandler) ListExecutions(c *gin.Context) {
	name := c.Param("name")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	executions, total, err := h.service.ListExecutions(c.Request.Context(), name, page, pageSize)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.SuccessWithData(c, gin.H{
		"total":      total,
		"executions": executions,
	})
}
