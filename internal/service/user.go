package service

import (
	"context"

	"github.com/samber/lo"
	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/internal/dal/model"
	"github.com/supuwoerc/gapi-server/pkg/logger"
	"github.com/supuwoerc/gapi-server/pkg/response"

	"go.uber.org/zap"
)

const maxCompletedTours = 50

type UserRepository interface {
	FindByID(ctx context.Context, id uint64) (*model.User, error)
	FindByIDWithRoles(ctx context.Context, id uint64) (*model.User, error)
	UpdateProfile(ctx context.Context, id uint64, username, bio, avatar string) error
	UpdateCompletedTours(ctx context.Context, id uint64, tours []string) error
}

type UserPermissionRepository interface {
	FindCodesByRoleIDsAndResourceType(ctx context.Context, roleIDs []uint64, resourceType model.ResourceType) ([]string, error)
}

type UserService struct {
	UserRepo UserRepository
	PermRepo UserPermissionRepository
	Config   *config.Config
	Logger   *logger.Logger
}

func (s *UserService) GetProfile(ctx context.Context, userID uint64) (*model.User, error) {
	user, err := s.UserRepo.FindByIDWithRoles(ctx, userID)
	if err != nil {
		s.Logger.Ctx(ctx).Error("find user with roles failed", zap.Error(err))
		return nil, response.InternalError
	}
	return user, nil
}

func (s *UserService) GetMenuPermissions(ctx context.Context, roleIDs []uint64) ([]string, error) {
	perms, err := s.PermRepo.FindCodesByRoleIDsAndResourceType(ctx, roleIDs, model.ResourceTypeFrontendMenu)
	if err != nil {
		s.Logger.Ctx(ctx).Error("find menu permissions failed", zap.Error(err))
		return nil, response.InternalError
	}
	return perms, nil
}

func (s *UserService) GetRoutePermissions(ctx context.Context, roleIDs []uint64) ([]string, error) {
	perms, err := s.PermRepo.FindCodesByRoleIDsAndResourceType(ctx, roleIDs, model.ResourceTypeFrontendRoute)
	if err != nil {
		s.Logger.Ctx(ctx).Error("find route permissions failed", zap.Error(err))
		return nil, response.InternalError
	}
	return perms, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID uint64, name, bio, avatar string) (*model.User, error) {
	if err := s.UserRepo.UpdateProfile(ctx, userID, name, bio, avatar); err != nil {
		s.Logger.Ctx(ctx).Error("update profile failed", zap.Uint64("userID", userID), zap.Error(err))
		return nil, response.InternalError
	}
	user, err := s.UserRepo.FindByID(ctx, userID)
	if err != nil {
		s.Logger.Ctx(ctx).Error("find user after profile update failed", zap.Uint64("userID", userID), zap.Error(err))
		return nil, response.InternalError
	}
	return user, nil
}

func (s *UserService) UpdateCompletedTours(ctx context.Context, userID uint64, newTours []string) ([]string, error) {
	validIDs := s.Config.Tour.ValidIDs
	if lo.SomeBy(newTours, func(t string) bool {
		return !lo.Contains(validIDs, t)
	}) {
		return nil, response.TourIDInvalid
	}
	user, err := s.UserRepo.FindByID(ctx, userID)
	if err != nil {
		s.Logger.Ctx(ctx).Error("find user failed", zap.Uint64("userID", userID), zap.Error(err))
		return nil, response.InternalError
	}
	merged := lo.Uniq(append([]string(user.CompletedTours), newTours...))
	if len(merged) > maxCompletedTours {
		return nil, response.TourLimitExceeded
	}
	if err := s.UserRepo.UpdateCompletedTours(ctx, userID, merged); err != nil {
		s.Logger.Ctx(ctx).Error("update completed tours failed", zap.Uint64("userID", userID), zap.Error(err))
		return nil, response.InternalError
	}
	return merged, nil
}
