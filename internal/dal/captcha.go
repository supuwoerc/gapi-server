package dal

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const captchaPrefix = "captcha:slide:"

type CaptchaDal struct {
	Redis *redis.Client
}

func (d *CaptchaDal) StoreCaptchaAnswer(ctx context.Context, captchaID string, x, y int, expiry time.Duration) error {
	key := captchaPrefix + captchaID
	pipe := d.Redis.TxPipeline()
	pipe.HSet(ctx, key, map[string]interface{}{
		"x": x,
		"y": y,
	})
	pipe.Expire(ctx, key, expiry)
	_, err := pipe.Exec(ctx)
	return err
}

func (d *CaptchaDal) GetCaptchaAnswer(ctx context.Context, captchaID string) (int, int, error) {
	key := captchaPrefix + captchaID
	result, err := d.Redis.HGetAll(ctx, key).Result()
	if err != nil {
		return 0, 0, err
	}
	if len(result) == 0 {
		return 0, 0, fmt.Errorf("captcha not found: %s", captchaID)
	}
	x, _ := strconv.Atoi(result["x"])
	y, _ := strconv.Atoi(result["y"])
	return x, y, nil
}

func (d *CaptchaDal) DeleteCaptchaAnswer(ctx context.Context, captchaID string) error {
	return d.Redis.Del(ctx, captchaPrefix+captchaID).Err()
}
