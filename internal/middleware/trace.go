package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"gapi-server/pkg/logger"

	"github.com/gin-gonic/gin"
)

const traceIDHeader = "X-Trace-ID"

func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(traceIDHeader)
		if traceID == "" {
			traceID = generateTraceID()
		}
		ctx := context.WithValue(c.Request.Context(), logger.TraceIDKey, traceID)
		c.Request = c.Request.WithContext(ctx)
		c.Header(traceIDHeader, traceID)
		c.Next()
	}
}

func generateTraceID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
