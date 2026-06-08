package captcha_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supuwoerc/gapi-server/internal/captcha"
)

func TestSlideCaptcha_Generate(t *testing.T) {
	sc, err := captcha.NewSlideCaptcha()
	require.NoError(t, err)

	data, err := sc.Generate()
	require.NoError(t, err)

	assert.NotEmpty(t, data.MasterImage)
	assert.NotEmpty(t, data.TileImage)
	assert.Greater(t, data.X, 0)
	assert.GreaterOrEqual(t, data.Y, 0)
}

func TestValidateSlide_Success(t *testing.T) {
	ok := captcha.ValidateSlide(100, 80, 102, 80, 5)
	assert.True(t, ok)
}

func TestValidateSlide_Fail(t *testing.T) {
	ok := captcha.ValidateSlide(100, 80, 200, 80, 5)
	assert.False(t, ok)
}
