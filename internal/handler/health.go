package handler

import (
	"gapi-server/pkg/logger"
	"gapi-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	logger *logger.Logger
}

func NewHealthHandler(logger *logger.Logger) *HealthHandler {
	return &HealthHandler{logger: logger}
}

func (h *HealthHandler) Check(c *gin.Context) {
	response.Success(c)
}
