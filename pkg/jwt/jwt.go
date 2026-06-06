package jwt

import (
	"fmt"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

type TokenType string

const (
	AccessTokenType  TokenType = "access"
	RefreshTokenType TokenType = "refresh"
)

type Claims struct {
	gojwt.RegisteredClaims
	UserID   uint64    `json:"uid"`
	Username string    `json:"username"`
	Type     TokenType `json:"type"`
}

type TokenPair struct {
	AccessToken  string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

type Manager struct {
	secret             []byte
	issuer             string
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

func NewManager(secret, issuer string, accessMinutes, refreshHours int) *Manager {
	return &Manager{
		secret:             []byte(secret),
		issuer:             issuer,
		accessTokenExpiry:  time.Duration(accessMinutes) * time.Minute,
		refreshTokenExpiry: time.Duration(refreshHours) * time.Hour,
	}
}

func (m *Manager) GenerateTokenPair(userID uint64, username string) (*TokenPair, error) {
	accessToken, err := m.generateToken(userID, username, AccessTokenType, m.accessTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	refreshToken, err := m.generateToken(userID, username, RefreshTokenType, m.refreshTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (m *Manager) ParseAccessToken(tokenStr string) (*Claims, error) {
	return m.parseToken(tokenStr, AccessTokenType)
}

func (m *Manager) ParseRefreshToken(tokenStr string) (*Claims, error) {
	return m.parseToken(tokenStr, RefreshTokenType)
}

func (m *Manager) RefreshTokenExpiry() time.Duration {
	return m.refreshTokenExpiry
}

func (m *Manager) generateToken(userID uint64, username string, tokenType TokenType, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := &Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			Issuer:    m.issuer,
			IssuedAt:  gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(expiry)),
		},
		UserID:   userID,
		Username: username,
		Type:     tokenType,
	}
	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *Manager) parseToken(tokenStr string, expectedType TokenType) (*Claims, error) {
	token, err := gojwt.ParseWithClaims(tokenStr, &Claims{}, func(t *gojwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.Type != expectedType {
		return nil, fmt.Errorf("token type mismatch: expected %s, got %s", expectedType, claims.Type)
	}
	return claims, nil
}
