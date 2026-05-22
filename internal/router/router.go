package router

import (
	"gapi-server/internal/handler"
	"gapi-server/internal/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewEngine sets up the gin engine with middleware and routes.
func NewEngine(logger *zap.Logger, healthHandler *handler.HealthHandler) *gin.Engine {
	r := gin.New()

	// Global middleware
	r.Use(gin.Recovery())
	r.Use(middleware.Logger(logger))

	// Health check
	r.GET("/health", healthHandler.Check)

	// API group placeholder
	_ = r.Group("/api/v1")

	return r
}
