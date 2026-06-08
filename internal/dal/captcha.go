package dal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wenlng/go-captcha/v2/click"
)

const (
	captchaSlidePrefix  = "captcha:slide:"
	captchaClickPrefix  = "captcha:click:"
	captchaRotatePrefix = "captcha:rotate:"
	captchaTokenPrefix  = "captcha:token:"
)

type CaptchaDal struct {
	Redis *redis.Client
}

// --- Slide ---

func (d *CaptchaDal) StoreSlideAnswer(ctx context.Context, captchaID string, x, y int, expiry time.Duration) error {
	key := captchaSlidePrefix + captchaID
	pipe := d.Redis.TxPipeline()
	pipe.HSet(ctx, key, map[string]any{
		"x": x,
		"y": y,
	})
	pipe.Expire(ctx, key, expiry)
	_, err := pipe.Exec(ctx)
	return err
}

func (d *CaptchaDal) GetSlideAnswer(ctx context.Context, captchaID string) (int, int, error) {
	key := captchaSlidePrefix + captchaID
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

func (d *CaptchaDal) DeleteSlideAnswer(ctx context.Context, captchaID string) error {
	return d.Redis.Del(ctx, captchaSlidePrefix+captchaID).Err()
}

// --- Click ---

func (d *CaptchaDal) StoreClickAnswer(ctx context.Context, captchaID string, dots map[int]*click.Dot, expiry time.Duration) error {
	data, err := json.Marshal(dots)
	if err != nil {
		return err
	}
	return d.Redis.Set(ctx, captchaClickPrefix+captchaID, data, expiry).Err()
}

func (d *CaptchaDal) GetClickAnswer(ctx context.Context, captchaID string) (map[int]*click.Dot, error) {
	data, err := d.Redis.Get(ctx, captchaClickPrefix+captchaID).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("captcha not found: %s", captchaID)
		}
		return nil, err
	}
	var dots map[int]*click.Dot
	if err := json.Unmarshal(data, &dots); err != nil {
		return nil, err
	}
	return dots, nil
}

func (d *CaptchaDal) DeleteClickAnswer(ctx context.Context, captchaID string) error {
	return d.Redis.Del(ctx, captchaClickPrefix+captchaID).Err()
}

// --- Rotate ---

func (d *CaptchaDal) StoreRotateAnswer(ctx context.Context, captchaID string, angle int, expiry time.Duration) error {
	key := captchaRotatePrefix + captchaID
	pipe := d.Redis.TxPipeline()
	pipe.HSet(ctx, key, map[string]any{
		"angle": angle,
	})
	pipe.Expire(ctx, key, expiry)
	_, err := pipe.Exec(ctx)
	return err
}

func (d *CaptchaDal) GetRotateAnswer(ctx context.Context, captchaID string) (int, error) {
	key := captchaRotatePrefix + captchaID
	result, err := d.Redis.HGetAll(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if len(result) == 0 {
		return 0, fmt.Errorf("captcha not found: %s", captchaID)
	}
	angle, _ := strconv.Atoi(result["angle"])
	return angle, nil
}

func (d *CaptchaDal) DeleteRotateAnswer(ctx context.Context, captchaID string) error {
	return d.Redis.Del(ctx, captchaRotatePrefix+captchaID).Err()
}

// --- Token ---

func (d *CaptchaDal) StoreCaptchaToken(ctx context.Context, token string, expiry time.Duration) error {
	return d.Redis.Set(ctx, captchaTokenPrefix+token, "1", expiry).Err()
}

func (d *CaptchaDal) ValidateAndDeleteCaptchaToken(ctx context.Context, token string) error {
	key := captchaTokenPrefix + token
	deleted, err := d.Redis.Del(ctx, key).Result()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return fmt.Errorf("captcha token not found or already used: %s", token)
	}
	return nil
}
