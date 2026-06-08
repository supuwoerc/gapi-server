package captcha_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supuwoerc/gapi-server/internal/captcha"
)

func TestClickCaptcha_Generate(t *testing.T) {
	cc, err := captcha.NewClickCaptcha()
	require.NoError(t, err)

	data, err := cc.Generate()
	require.NoError(t, err)

	assert.NotEmpty(t, data.MasterImage)
	assert.NotEmpty(t, data.ThumbImage)
	assert.NotEmpty(t, data.Dots)
}

func TestValidateClick_Success(t *testing.T) {
	ok := captcha.ValidateClick(100, 80, 98, 78, 30, 30, 5)
	assert.True(t, ok)
}

func TestValidateClick_Fail(t *testing.T) {
	ok := captcha.ValidateClick(200, 200, 50, 50, 30, 30, 5)
	assert.False(t, ok)
}
