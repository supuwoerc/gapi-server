package middleware

import (
	"strings"

	"github.com/supuwoerc/gapi-server/pkg/jwt"
	"github.com/supuwoerc/gapi-server/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	AuthHeaderKey = "Authorization"
	BearerPrefix  = "Bearer "

	contextUserID   = "middleware.auth.user_id"
	contextUsername = "middleware.auth.username"
)

func CurrentUserID(c *gin.Context) (uint64, bool) {
	v, exists := c.Get(contextUserID)
	if !exists {
		return 0, false
	}
	id, ok := v.(uint64)
	return id, ok
}

func CurrentUsername(c *gin.Context) (string, bool) {
	v, exists := c.Get(contextUsername)
	if !exists {
		return "", false
	}
	name, ok := v.(string)
	return name, ok
}

func JWTAuth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader(AuthHeaderKey)
		if header == "" || !strings.HasPrefix(header, BearerPrefix) {
			response.FailWithCode(c, response.InvalidToken)
			return
		}
		tokenStr := strings.TrimPrefix(header, BearerPrefix)
		claims, err := jwtManager.ParseAccessToken(tokenStr)
		if err != nil {
			response.FailWithCode(c, response.InvalidToken)
			return
		}
		c.Set(contextUserID, claims.UserID)
		c.Set(contextUsername, claims.Username)
		c.Next()
	}
}
