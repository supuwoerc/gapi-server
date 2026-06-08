package dal_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supuwoerc/gapi-server/internal/dal"
)

func setupMiniredis(t *testing.T) *redis.Client {
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestCaptchaDal_StoreAndGet(t *testing.T) {
	client := setupMiniredis(t)
	d := &dal.CaptchaDal{Redis: client}
	ctx := context.Background()

	err := d.StoreCaptchaAnswer(ctx, "test-id-1", 150, 80, 2*time.Minute)
	require.NoError(t, err)

	x, y, err := d.GetCaptchaAnswer(ctx, "test-id-1")
	require.NoError(t, err)
	assert.Equal(t, 150, x)
	assert.Equal(t, 80, y)
}

func TestCaptchaDal_Delete(t *testing.T) {
	client := setupMiniredis(t)
	d := &dal.CaptchaDal{Redis: client}
	ctx := context.Background()

	_ = d.StoreCaptchaAnswer(ctx, "test-id-2", 100, 50, 2*time.Minute)
	err := d.DeleteCaptchaAnswer(ctx, "test-id-2")
	require.NoError(t, err)

	_, _, err = d.GetCaptchaAnswer(ctx, "test-id-2")
	assert.Error(t, err)
}

func TestCaptchaDal_GetNotExist(t *testing.T) {
	client := setupMiniredis(t)
	d := &dal.CaptchaDal{Redis: client}
	ctx := context.Background()

	_, _, err := d.GetCaptchaAnswer(ctx, "not-exist")
	assert.Error(t, err)
}
