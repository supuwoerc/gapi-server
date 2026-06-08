package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/supuwoerc/gapi-server/internal/captcha"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/resp"
	"github.com/supuwoerc/gapi-server/pkg/logger"
	"github.com/supuwoerc/gapi-server/pkg/response"

	"go.uber.org/zap"
)

const captchaExpiry = 3 * time.Minute

type CaptchaRepository interface {
	StoreCaptchaAnswer(ctx context.Context, captchaID string, x, y int, expiry time.Duration) error
	GetCaptchaAnswer(ctx context.Context, captchaID string) (int, int, error)
	DeleteCaptchaAnswer(ctx context.Context, captchaID string) error
}

type CaptchaService struct {
	Slide       *captcha.SlideCaptcha
	CaptchaRepo CaptchaRepository
	Logger      *logger.Logger
}

func (s *CaptchaService) GenerateSlideCaptcha(ctx context.Context) (*resp.CaptchaResponse, error) {
	data, err := s.Slide.Generate()
	if err != nil {
		s.Logger.Ctx(ctx).Error("generate slide captcha failed", zap.Error(err))
		return nil, response.CaptchaGenFailed
	}

	captchaID := uuid.New().String()
	if err := s.CaptchaRepo.StoreCaptchaAnswer(ctx, captchaID, data.X, data.Y, captchaExpiry); err != nil {
		s.Logger.Ctx(ctx).Error("store captcha answer failed", zap.Error(err))
		return nil, response.CaptchaGenFailed
	}

	return &resp.CaptchaResponse{
		CaptchaID:   captchaID,
		MasterImage: data.MasterImage,
		TileImage:   data.TileImage,
		TileY:       data.Y,
	}, nil
}

func (s *CaptchaService) ValidateSlideCaptcha(ctx context.Context, captchaID string, x, y int) (bool, error) {
	targetX, targetY, err := s.CaptchaRepo.GetCaptchaAnswer(ctx, captchaID)
	if err != nil {
		s.Logger.Ctx(ctx).Warn("get captcha answer failed", zap.String("captchaID", captchaID), zap.Error(err))
		return false, response.CaptchaExpired
	}

	if err := s.CaptchaRepo.DeleteCaptchaAnswer(ctx, captchaID); err != nil {
		s.Logger.Ctx(ctx).Error("delete captcha answer failed", zap.String("captchaID", captchaID), zap.Error(err))
	}

	ok := captcha.ValidateSlide(x, y, targetX, targetY, 5)
	return ok, nil
}
