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
	"github.com/wenlng/go-captcha/v2/click"
)

func setupMiniredis(t *testing.T) *redis.Client {
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

// --- Slide ---

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

// --- Click ---

func TestCaptchaDal_ClickStoreAndGet(t *testing.T) {
	client := setupMiniredis(t)
	d := &dal.CaptchaDal{Redis: client}
	ctx := context.Background()

	dots := map[int]*click.Dot{
		0: {X: 100, Y: 80, Width: 30, Height: 30},
		1: {X: 200, Y: 150, Width: 30, Height: 30},
	}

	err := d.StoreClickAnswer(ctx, "click-1", dots, 2*time.Minute)
	require.NoError(t, err)

	got, err := d.GetClickAnswer(ctx, "click-1")
	require.NoError(t, err)
	assert.Equal(t, 2, len(got))
	assert.Equal(t, 100, got[0].X)
	assert.Equal(t, 150, got[1].Y)
}

func TestCaptchaDal_ClickDelete(t *testing.T) {
	client := setupMiniredis(t)
	d := &dal.CaptchaDal{Redis: client}
	ctx := context.Background()

	dots := map[int]*click.Dot{
		0: {X: 50, Y: 60, Width: 30, Height: 30},
	}
	_ = d.StoreClickAnswer(ctx, "click-2", dots, 2*time.Minute)
	err := d.DeleteClickAnswer(ctx, "click-2")
	require.NoError(t, err)

	_, err = d.GetClickAnswer(ctx, "click-2")
	assert.Error(t, err)
}

func TestCaptchaDal_ClickGetNotExist(t *testing.T) {
	client := setupMiniredis(t)
	d := &dal.CaptchaDal{Redis: client}
	ctx := context.Background()

	_, err := d.GetClickAnswer(ctx, "not-exist")
	assert.Error(t, err)
}

// --- Rotate ---

func TestCaptchaDal_RotateStoreAndGet(t *testing.T) {
	client := setupMiniredis(t)
	d := &dal.CaptchaDal{Redis: client}
	ctx := context.Background()

	err := d.StoreRotateAnswer(ctx, "rotate-1", 45, 2*time.Minute)
	require.NoError(t, err)

	angle, err := d.GetRotateAnswer(ctx, "rotate-1")
	require.NoError(t, err)
	assert.Equal(t, 45, angle)
}

func TestCaptchaDal_RotateDelete(t *testing.T) {
	client := setupMiniredis(t)
	d := &dal.CaptchaDal{Redis: client}
	ctx := context.Background()

	_ = d.StoreRotateAnswer(ctx, "rotate-2", 90, 2*time.Minute)
	err := d.DeleteRotateAnswer(ctx, "rotate-2")
	require.NoError(t, err)

	_, err = d.GetRotateAnswer(ctx, "rotate-2")
	assert.Error(t, err)
}

func TestCaptchaDal_RotateGetNotExist(t *testing.T) {
	client := setupMiniredis(t)
	d := &dal.CaptchaDal{Redis: client}
	ctx := context.Background()

	_, err := d.GetRotateAnswer(ctx, "not-exist")
	assert.Error(t, err)
}

// --- Token ---

func TestCaptchaDal_TokenStoreAndValidate(t *testing.T) {
	client := setupMiniredis(t)
	d := &dal.CaptchaDal{Redis: client}
	ctx := context.Background()

	err := d.StoreCaptchaToken(ctx, "token-abc", 5*time.Minute)
	require.NoError(t, err)

	err = d.ValidateAndDeleteCaptchaToken(ctx, "token-abc")
	require.NoError(t, err)

	// second validation should fail (one-time use)
	err = d.ValidateAndDeleteCaptchaToken(ctx, "token-abc")
	assert.Error(t, err)
}

func TestCaptchaDal_TokenNotExist(t *testing.T) {
	client := setupMiniredis(t)
	d := &dal.CaptchaDal{Redis: client}
	ctx := context.Background()

	err := d.ValidateAndDeleteCaptchaToken(ctx, "no-such-token")
	assert.Error(t, err)
}
