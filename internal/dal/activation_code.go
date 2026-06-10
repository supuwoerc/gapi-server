package dal

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const activationCodePrefix = "email:activation:"

type ActivationCodeDal struct {
	Redis *redis.Client
}

func (d *ActivationCodeDal) StoreActivationCode(ctx context.Context, email, code string, expiry time.Duration) error {
	return d.Redis.Set(ctx, d.key(email), code, expiry).Err()
}

func (d *ActivationCodeDal) GetActivationCode(ctx context.Context, email string) (string, error) {
	return d.Redis.Get(ctx, d.key(email)).Result()
}

func (d *ActivationCodeDal) DeleteActivationCode(ctx context.Context, email string) error {
	return d.Redis.Del(ctx, d.key(email)).Err()
}

func (d *ActivationCodeDal) key(email string) string {
	return fmt.Sprintf("%s%s", activationCodePrefix, email)
}
