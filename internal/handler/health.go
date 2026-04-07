package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HealthHandler handles health-check endpoints.
type HealthHandler struct {
	logger *zap.Logger
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(logger *zap.Logger) *HealthHandler {
	return &HealthHandler{logger: logger}
}

// Check returns a simple health status.
func (h *HealthHandler) Check(c *gin.Context) {
	h.logger.Info("health check called")
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
