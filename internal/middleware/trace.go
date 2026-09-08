package middleware

import (
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/gin-gonic/gin"
)

const traceIDHeader = "X-Trace-ID"

func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(traceIDHeader)
		if traceID == "" {
			traceID = logger.GenerateTraceID()
		}
		c.Request = c.Request.WithContext(logger.WithTraceID(c.Request.Context(), traceID))
		c.Header(traceIDHeader, traceID)
		c.Next()
	}
}
