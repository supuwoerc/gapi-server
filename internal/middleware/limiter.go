package middleware

import (
	"gapi-server/internal/config"
	"gapi-server/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	limiter "github.com/ulule/limiter/v3"
	ginLimiter "github.com/ulule/limiter/v3/drivers/middleware/gin"
	storeRedis "github.com/ulule/limiter/v3/drivers/store/redis"
)

func RateLimit(cfg *config.RateLimitConfig, client *redis.Client) gin.HandlerFunc {
	rate, err := limiter.NewRateFromFormatted(cfg.Pattern)
	if err != nil {
		panic("invalid rate limit pattern: " + cfg.Pattern)
	}
	store, err := storeRedis.NewStoreWithOptions(client, limiter.StoreOptions{
		Prefix: cfg.Prefix,
	})
	if err != nil {
		panic("failed to create rate limit store: " + err.Error())
	}
	instance := limiter.New(store, rate)
	return ginLimiter.NewMiddleware(instance,
		ginLimiter.WithErrorHandler(func(c *gin.Context, err error) {
			response.FailWithError(c, err)
			c.Abort()
		}),
		ginLimiter.WithLimitReachedHandler(func(c *gin.Context) {
			response.FailWithCode(c, response.Busy)
			c.Abort()
		}),
	)
}
