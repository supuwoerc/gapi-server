package router

import (
	"fmt"

	"gapi-server/internal/config"
	"gapi-server/internal/handler"
	"gapi-server/internal/middleware"
	"gapi-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func NewEngine(l *logger.Logger, cfg *config.Config, redisClient *redis.Client, healthHandler *handler.HealthHandler) *gin.Engine {
	gin.DebugPrintFunc = func(format string, values ...interface{}) {
		l.Debug(fmt.Sprintf(format, values...))
	}
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
		l.Debug("route registered",
			zap.String("method", httpMethod),
			zap.String("path", absolutePath),
			zap.String("handler", handlerName),
		)
	}

	r := gin.New()
	r.ForwardedByClientIP = true
	if cfg.Server.MaxMultipartMemory > 0 {
		r.MaxMultipartMemory = cfg.Server.MaxMultipartMemory << 20
	}

	r.Use(middleware.Recovery(l))
	r.Use(middleware.Logger(l))
	r.Use(middleware.Cors(&cfg.Cors))
	r.Use(middleware.RateLimit(&cfg.RateLimit, redisClient))

	r.GET("/health", healthHandler.Check)

	_ = r.Group("/api/v1")

	return r
}
