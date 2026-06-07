package dal

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const refreshTokenPrefix = "auth:refresh:"

type TokenDal struct {
	Redis *redis.Client
}

func (d *TokenDal) StoreRefreshToken(ctx context.Context, userID uint64, token string, expiry time.Duration) error {
	return d.Redis.Set(ctx, d.key(userID), token, expiry).Err()
}

func (d *TokenDal) GetRefreshToken(ctx context.Context, userID uint64) (string, error) {
	return d.Redis.Get(ctx, d.key(userID)).Result()
}

func (d *TokenDal) DeleteRefreshToken(ctx context.Context, userID uint64) error {
	return d.Redis.Del(ctx, d.key(userID)).Err()
}

func (d *TokenDal) key(userID uint64) string {
	return fmt.Sprintf("%s%d", refreshTokenPrefix, userID)
}
