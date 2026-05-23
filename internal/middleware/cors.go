package middleware

import (
	"strings"
	"time"

	"github.com/supuwoerc/gapi-server/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Cors(cfg *config.CorsConfig) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			for _, prefix := range cfg.OriginPrefixes {
				if strings.HasPrefix(origin, prefix) {
					return true
				}
			}
			return false
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "Locale", "Refresh-Token"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
