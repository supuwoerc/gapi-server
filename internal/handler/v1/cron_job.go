package v1

import (
	stderrors "errors"

	"github.com/supuwoerc/gapi-server/internal/cronjob"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/req"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/resp"
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
		jobs.PUT("/:name/enable", h.Enable)
		jobs.PUT("/:name/disable", h.Disable)
		jobs.POST("/:name/trigger", h.Trigger)
		jobs.GET("/:name/executions", h.ListExecutions)
	}
}

// List
// @Summary      列出所有定时任务
// @Description  返回所有已注册的定时任务及其状态
// @Tags         定时任务
// @Produce      json
// @Success      200  {object}  response.BasicResponse[resp.CronJobListResponse]
// @Failure      500  {object}  response.Response
// @Router       /api/v1/cron-jobs [get]
func (h *CronJobHandler) List(c *gin.Context) {
	jobs, err := h.service.ListJobs(c.Request.Context())
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.SuccessWithData(c, resp.CronJobListResponse{Jobs: jobs})
}

// Get
// @Summary      获取定时任务详情
// @Description  根据名称获取定时任务详情
// @Tags         定时任务
// @Produce      json
// @Param        name  path  string  true  "任务名称"
// @Success      200  {object}  response.BasicResponse[resp.CronJobDetailResponse]
// @Failure      500  {object}  response.Response
// @Router       /api/v1/cron-jobs/{name} [get]
func (h *CronJobHandler) Get(c *gin.Context) {
	var uri req.CronJobUriRequest
	if err := c.ShouldBindUri(&uri); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	job, err := h.service.GetJob(c.Request.Context(), uri.Name)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.SuccessWithData(c, resp.CronJobDetailResponse{CronJob: job})
}

// Enable
// @Summary      启用定时任务
// @Description  启用指定的定时任务
// @Tags         定时任务
// @Produce      json
// @Param        name  path  string  true  "任务名称"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /api/v1/cron-jobs/{name}/enable [put]
func (h *CronJobHandler) Enable(c *gin.Context) {
	var uri req.CronJobUriRequest
	if err := c.ShouldBindUri(&uri); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	ctx := c.Request.Context()
	if err := h.service.SetEnabled(ctx, uri.Name, true); err != nil {
		response.FailWithError(c, err)
		return
	}
	if err := h.jobManager.EnableJob(ctx, uri.Name); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c)
}

// Disable
// @Summary      禁用定时任务
// @Description  禁用指定的定时任务，可通过 force 参数立即取消正在执行的任务
// @Tags         定时任务
// @Produce      json
// @Param        name   path   string  true   "任务名称"
// @Param        force  query  bool    false  "是否立即取消正在执行的任务"  default(false)
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /api/v1/cron-jobs/{name}/disable [put]
func (h *CronJobHandler) Disable(c *gin.Context) {
	var uri req.CronJobUriRequest
	if err := c.ShouldBindUri(&uri); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	var query req.CronJobDisableRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	ctx := c.Request.Context()
	if err := h.service.SetEnabled(ctx, uri.Name, false); err != nil {
		response.FailWithError(c, err)
		return
	}
	if err := h.jobManager.DisableJob(ctx, uri.Name, query.Force); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c)
}

// Trigger
// @Summary      手动触发定时任务
// @Description  立即执行一次指定的定时任务，可通过 force 参数强制执行（忽略正在运行的检查）
// @Tags         定时任务
// @Produce      json
// @Param        name   path   string  true   "任务名称"
// @Param        force  query  bool    false  "是否强制执行"  default(false)
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /api/v1/cron-jobs/{name}/trigger [post]
func (h *CronJobHandler) Trigger(c *gin.Context) {
	var uri req.CronJobUriRequest
	if err := c.ShouldBindUri(&uri); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	var query req.CronJobTriggerRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	if err := h.jobManager.TriggerManual(c.Request.Context(), uri.Name, query.Force); err != nil {
		if stderrors.Is(err, cronjob.ErrJobRunning) {
			response.FailWithCode(c, response.Busy)
			return
		}
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
// @Param        page       query  int     false  "页码"      default(1)
// @Param        page_size  query  int     false  "每页数量"  default(20)
// @Success      200  {object}  response.BasicResponse[resp.CronJobListExecutionsResponse]
// @Failure      400  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Router       /api/v1/cron-jobs/{name}/executions [get]
func (h *CronJobHandler) ListExecutions(c *gin.Context) {
	var uri req.CronJobUriRequest
	if err := c.ShouldBindUri(&uri); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	var query req.CronJobListExecutionsRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	executions, total, err := h.service.ListExecutions(c.Request.Context(), uri.Name, query.Page, query.PageSize)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.SuccessWithData(c, resp.CronJobListExecutionsResponse{
		Total:      total,
		Executions: executions,
	})
}
