package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndParseAccessToken(t *testing.T) {
	m := NewManager("test-secret", "gapi", 15, 168)
	pair, err := m.GenerateTokenPair(1, "testuser")
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)

	claims, err := m.ParseAccessToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
}

func TestParseExpiredAccessToken(t *testing.T) {
	m := NewManager("test-secret", "gapi", -1, 168)
	pair, err := m.GenerateTokenPair(1, "testuser")
	require.NoError(t, err)

	_, err = m.ParseAccessToken(pair.AccessToken)
	assert.Error(t, err)
}

func TestParseRefreshToken(t *testing.T) {
	m := NewManager("test-secret", "gapi", 15, 168)
	pair, err := m.GenerateTokenPair(1, "testuser")
	require.NoError(t, err)

	claims, err := m.ParseRefreshToken(pair.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), claims.UserID)
}

func TestInvalidSecret(t *testing.T) {
	m1 := NewManager("secret-1", "gapi", 15, 168)
	m2 := NewManager("secret-2", "gapi", 15, 168)

	pair, err := m1.GenerateTokenPair(1, "testuser")
	require.NoError(t, err)

	_, err = m2.ParseAccessToken(pair.AccessToken)
	assert.Error(t, err)
}

func TestRefreshTokenExpiry(t *testing.T) {
	m := NewManager("test-secret", "gapi", 15, 168)
	pair, err := m.GenerateTokenPair(1, "testuser")
	require.NoError(t, err)

	claims, err := m.ParseRefreshToken(pair.RefreshToken)
	require.NoError(t, err)
	expiry := claims.ExpiresAt.Time
	assert.True(t, expiry.After(time.Now().Add(167*time.Hour)))
	assert.True(t, expiry.Before(time.Now().Add(169*time.Hour)))
}

func TestTokenTypeMismatch(t *testing.T) {
	m := NewManager("test-secret", "gapi", 15, 168)
	pair, err := m.GenerateTokenPair(1, "testuser")
	require.NoError(t, err)

	_, err = m.ParseRefreshToken(pair.AccessToken)
	assert.Error(t, err)

	_, err = m.ParseAccessToken(pair.RefreshToken)
	assert.Error(t, err)
}
