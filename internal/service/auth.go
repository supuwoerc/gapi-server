package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/supuwoerc/gapi-server/internal/dal/model"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/resp"
	"github.com/supuwoerc/gapi-server/pkg/jwt"
	"github.com/supuwoerc/gapi-server/pkg/logger"
	"github.com/supuwoerc/gapi-server/pkg/response"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	refreshTokenPrefix = "auth:refresh:"
	maxLoginFailCount  = 5
	lockDuration       = 30 * time.Minute
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	FindByID(ctx context.Context, id uint64) (*model.User, error)
	FindByIDWithRoles(ctx context.Context, id uint64) (*model.User, error)
	UpdateLastLogin(ctx context.Context, id uint64) error
	IncrementLoginFail(ctx context.Context, id uint64) error
	LockUser(ctx context.Context, id uint64, until time.Time) error
}

type AuthService struct {
	UserRepo    UserRepository
	JWTManager  *jwt.Manager
	RedisClient *redis.Client
	Logger      *logger.Logger
}

func (s *AuthService) Register(ctx context.Context, username, email, password string) error {
	if _, err := s.UserRepo.FindByEmail(ctx, email); err == nil {
		return response.UserAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.Logger.Ctx(ctx).Error("find user by email failed", zap.Error(err))
		return response.InternalError
	}

	if _, err := s.UserRepo.FindByUsername(ctx, username); err == nil {
		return response.UserAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.Logger.Ctx(ctx).Error("find user by username failed", zap.Error(err))
		return response.InternalError
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.Logger.Ctx(ctx).Error("hash password failed", zap.Error(err))
		return response.InternalError
	}

	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Status:       1,
	}
	if err := s.UserRepo.Create(ctx, user); err != nil {
		s.Logger.Ctx(ctx).Error("create user failed", zap.Error(err))
		return response.InternalError
	}
	return nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*jwt.TokenPair, *model.User, error) {
	user, err := s.UserRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, response.InvalidCredential
		}
		s.Logger.Ctx(ctx).Error("find user failed", zap.Error(err))
		return nil, nil, response.InternalError
	}

	if user.Status == 0 {
		return nil, nil, response.UserDisabled
	}

	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, nil, response.UserLocked
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		_ = s.UserRepo.IncrementLoginFail(ctx, user.ID)
		if user.LoginFailCount+1 >= maxLoginFailCount {
			_ = s.UserRepo.LockUser(ctx, user.ID, time.Now().Add(lockDuration))
		}
		return nil, nil, response.InvalidCredential
	}

	_ = s.UserRepo.UpdateLastLogin(ctx, user.ID)

	pair, err := s.JWTManager.GenerateTokenPair(user.ID, user.Username)
	if err != nil {
		s.Logger.Ctx(ctx).Error("generate token pair failed", zap.Error(err))
		return nil, nil, response.InternalError
	}

	s.storeRefreshToken(ctx, user.ID, pair.RefreshToken)

	return pair, user, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	claims, err := s.JWTManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, response.TokenExpired
	}

	key := s.refreshTokenKey(claims.UserID)
	stored, err := s.RedisClient.Get(ctx, key).Result()
	if err != nil || stored != refreshToken {
		return nil, response.RefreshTokenUsed
	}

	pair, err := s.JWTManager.GenerateTokenPair(claims.UserID, claims.Username)
	if err != nil {
		s.Logger.Ctx(ctx).Error("generate token pair failed", zap.Error(err))
		return nil, response.InternalError
	}

	s.storeRefreshToken(ctx, claims.UserID, pair.RefreshToken)

	return pair, nil
}

func (s *AuthService) Logout(ctx context.Context, userID uint64) {
	key := s.refreshTokenKey(userID)
	s.RedisClient.Del(ctx, key)
}

func (s *AuthService) GetProfile(ctx context.Context, userID uint64) (*resp.LoginResponse, error) {
	user, err := s.UserRepo.FindByIDWithRoles(ctx, userID)
	if err != nil {
		s.Logger.Ctx(ctx).Error("find user with roles failed", zap.Error(err))
		return nil, response.InternalError
	}
	roles := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roles = append(roles, r.Code)
	}
	return &resp.LoginResponse{
		User: resp.UserInfo{
			Name:   user.Username,
			Email:  user.Email,
			Avatar: user.Avatar,
			Bio:    "",
		},
		Role:             roles,
		MenuPermissions:  []string{},
		RoutePermissions: []string{},
		CompletedTours:   []string{},
	}, nil
}

func (s *AuthService) storeRefreshToken(ctx context.Context, userID uint64, token string) {
	key := s.refreshTokenKey(userID)
	s.RedisClient.Set(ctx, key, token, s.JWTManager.RefreshTokenExpiry())
}

func (s *AuthService) refreshTokenKey(userID uint64) string {
	return fmt.Sprintf("%s%d", refreshTokenPrefix, userID)
}
