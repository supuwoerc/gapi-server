package router

import (
	"fmt"

	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/internal/middleware"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func NewEngine(l *logger.Logger, cfg *config.Config, redisClient *redis.Client, h *V1Handlers) *gin.Engine {
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

	r.Use(middleware.Trace())
	r.Use(middleware.I18n(&cfg.Locale))
	r.Use(middleware.Validator(&cfg.Locale))
	r.Use(middleware.Recovery(l))
	r.Use(middleware.Logger(l))
	r.Use(middleware.Cors(&cfg.Cors))
	r.Use(middleware.RateLimit(&cfg.RateLimit, redisClient))

	v1Route := r.Group("/api/v1")

	h.Register(v1Route)
	initSwagger(v1Route, cfg.Env)

	return r
}
