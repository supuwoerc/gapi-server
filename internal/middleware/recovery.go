package middleware

import (
	"runtime/debug"

	"gapi-server/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, err any) {
		stack := debug.Stack()
		logger.Error("panic recovered",
			zap.Any("error", err),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.ByteString("stack", stack),
		)
		response.FailWithCode(c, response.RecoveryError)
		c.Abort()
	})
}
