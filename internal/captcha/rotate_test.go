package captcha_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supuwoerc/gapi-server/internal/captcha"
)

func TestRotateCaptcha_Generate(t *testing.T) {
	rc, err := captcha.NewRotateCaptcha()
	require.NoError(t, err)

	data, err := rc.Generate()
	require.NoError(t, err)

	assert.NotEmpty(t, data.MasterImage)
	assert.NotEmpty(t, data.ThumbImage)
	assert.NotZero(t, data.Angle)
}

func TestValidateRotate_Success(t *testing.T) {
	ok := captcha.ValidateRotate(330, 30, 5)
	assert.True(t, ok)
}

func TestValidateRotate_Fail(t *testing.T) {
	ok := captcha.ValidateRotate(100, 30, 5)
	assert.False(t, ok)
}
