package service

import (
	"context"
	"errors"
	"time"

	"github.com/supuwoerc/gapi-server/internal/dal/model"
	"github.com/supuwoerc/gapi-server/pkg/database"
	"github.com/supuwoerc/gapi-server/pkg/jwt"
	"github.com/supuwoerc/gapi-server/pkg/logger"
	"github.com/supuwoerc/gapi-server/pkg/response"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	maxLoginFailCount = 5
	lockDuration      = 30 * time.Minute
)

type AuthUserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByEmailWithRoles(ctx context.Context, email string) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	FindByID(ctx context.Context, id uint64) (*model.User, error)
	FindByIDWithRoles(ctx context.Context, id uint64) (*model.User, error)
	UpdateLastLogin(ctx context.Context, id uint64) error
	IncrementLoginFail(ctx context.Context, id uint64) error
	LockUser(ctx context.Context, id uint64, until time.Time) error
}

type TokenRepository interface {
	StoreRefreshToken(ctx context.Context, userID uint64, token string, expiry time.Duration) error
	GetRefreshToken(ctx context.Context, userID uint64) (string, error)
	DeleteRefreshToken(ctx context.Context, userID uint64) error
}

type AuthPermissionRepository interface {
	FindCodesByRoleIDsAndResourceType(ctx context.Context, roleIDs []uint64, resourceType model.ResourceType) ([]string, error)
	FindCodesByRoleIDsAndModule(ctx context.Context, roleIDs []uint64, module string) ([]string, error)
}

type AuthService struct {
	UserRepo   AuthUserRepository
	TokenRepo  TokenRepository
	PermRepo   AuthPermissionRepository
	TxManager  *database.TransactionManager
	JWTManager *jwt.Manager
	Logger     *logger.Logger
}

func (s *AuthService) Register(ctx context.Context, username, email, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.Logger.Ctx(ctx).Error("hash password failed", zap.Error(err))
		return response.InternalError
	}

	return s.TxManager.Transaction(ctx, func(txCtx context.Context) error {
		if _, err := s.UserRepo.FindByEmail(txCtx, email); err == nil {
			return response.UserAlreadyExists
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			s.Logger.Ctx(txCtx).Error("find user by email failed", zap.Error(err))
			return response.InternalError
		}

		if _, err := s.UserRepo.FindByUsername(txCtx, username); err == nil {
			return response.UserAlreadyExists
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			s.Logger.Ctx(txCtx).Error("find user by username failed", zap.Error(err))
			return response.InternalError
		}

		user := &model.User{
			Username:     username,
			Email:        email,
			PasswordHash: string(hash),
			Status:       1,
		}
		if err := s.UserRepo.Create(txCtx, user); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return response.UserAlreadyExists
			}
			s.Logger.Ctx(txCtx).Error("create user failed", zap.Error(err))
			return response.InternalError
		}
		return nil
	})
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*jwt.TokenPair, *model.User, error) {
	user, err := s.UserRepo.FindByEmailWithRoles(ctx, email)
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
		if err := s.UserRepo.IncrementLoginFail(ctx, user.ID); err != nil {
			s.Logger.Ctx(ctx).Error("increment login fail count failed", zap.Uint64("userID", user.ID), zap.Error(err))
		}
		if user.LoginFailCount+1 >= maxLoginFailCount {
			if err := s.UserRepo.LockUser(ctx, user.ID, time.Now().Add(lockDuration)); err != nil {
				s.Logger.Ctx(ctx).Error("lock user failed", zap.Uint64("userID", user.ID), zap.Error(err))
			}
		}
		return nil, nil, response.InvalidCredential
	}

	if err := s.UserRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.Logger.Ctx(ctx).Error("update last login failed", zap.Uint64("userID", user.ID), zap.Error(err))
	}

	pair, err := s.JWTManager.GenerateTokenPair(user.ID, user.Username)
	if err != nil {
		s.Logger.Ctx(ctx).Error("generate token pair failed", zap.Error(err))
		return nil, nil, response.InternalError
	}

	if err := s.TokenRepo.StoreRefreshToken(ctx, user.ID, pair.RefreshToken, s.JWTManager.RefreshTokenExpiry()); err != nil {
		s.Logger.Ctx(ctx).Error("store refresh token failed", zap.Error(err))
		return nil, nil, response.InternalError
	}

	return pair, user, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	claims, err := s.JWTManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, response.TokenExpired
	}

	stored, err := s.TokenRepo.GetRefreshToken(ctx, claims.UserID)
	if err != nil || stored != refreshToken {
		return nil, response.RefreshTokenUsed
	}

	pair, err := s.JWTManager.GenerateTokenPair(claims.UserID, claims.Username)
	if err != nil {
		s.Logger.Ctx(ctx).Error("generate token pair failed", zap.Error(err))
		return nil, response.InternalError
	}

	if err := s.TokenRepo.StoreRefreshToken(ctx, claims.UserID, pair.RefreshToken, s.JWTManager.RefreshTokenExpiry()); err != nil {
		s.Logger.Ctx(ctx).Error("store refresh token failed", zap.Error(err))
		return nil, response.InternalError
	}

	return pair, nil
}

func (s *AuthService) Logout(ctx context.Context, userID uint64) {
	if err := s.TokenRepo.DeleteRefreshToken(ctx, userID); err != nil {
		s.Logger.Ctx(ctx).Error("delete refresh token failed", zap.Uint64("userID", userID), zap.Error(err))
	}
}

func (s *AuthService) GetPermissionsForRoles(ctx context.Context, roleIDs []uint64) ([]string, []string, error) {
	menuPerms, err := s.PermRepo.FindCodesByRoleIDsAndResourceType(ctx, roleIDs, model.ResourceTypeFrontendMenu)
	if err != nil {
		s.Logger.Ctx(ctx).Error("find menu permissions failed", zap.Error(err))
		return nil, nil, response.InternalError
	}
	routePerms, err := s.PermRepo.FindCodesByRoleIDsAndResourceType(ctx, roleIDs, model.ResourceTypeFrontendRoute)
	if err != nil {
		s.Logger.Ctx(ctx).Error("find route permissions failed", zap.Error(err))
		return nil, nil, response.InternalError
	}
	return menuPerms, routePerms, nil
}

func (s *AuthService) GetModulePermissions(ctx context.Context, roleIDs []uint64, module string) ([]string, error) {
	perms, err := s.PermRepo.FindCodesByRoleIDsAndModule(ctx, roleIDs, module)
	if err != nil {
		s.Logger.Ctx(ctx).Error("find module permissions failed", zap.String("module", module), zap.Error(err))
		return nil, response.InternalError
	}
	return perms, nil
}

func (s *AuthService) GetUserWithRoles(ctx context.Context, userID uint64) (*model.User, error) {
	user, err := s.UserRepo.FindByIDWithRoles(ctx, userID)
	if err != nil {
		s.Logger.Ctx(ctx).Error("find user with roles failed", zap.Error(err))
		return nil, response.InternalError
	}
	return user, nil
}
