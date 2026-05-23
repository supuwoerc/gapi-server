package v1

import (
	"github.com/supuwoerc/gapi-server/pkg/logger"
	"github.com/supuwoerc/gapi-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	logger *logger.Logger
}

func NewHealthHandler(logger *logger.Logger) *HealthHandler {
	return &HealthHandler{logger: logger}
}

// Register registers health routes on the given router group.
func (h *HealthHandler) Register(r *gin.RouterGroup) {
	r.GET("/health", h.Check)
}

// Check
// @Summary      健康检查
// @Description  返回服务运行状态
// @Tags         系统
// @Produce      json
// @Success      200  {object}  response.Response
// @Router       /api/v1/health [get]
func (h *HealthHandler) Check(c *gin.Context) {
	response.Success(c)
}
