package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/supuwoerc/gapi-server/internal/captcha"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/resp"
	"github.com/supuwoerc/gapi-server/pkg/logger"
	"github.com/supuwoerc/gapi-server/pkg/response"
	"github.com/wenlng/go-captcha/v2/click"

	"go.uber.org/zap"
)

const (
	captchaExpiry = 3 * time.Minute
	tokenExpiry   = 5 * time.Minute
)

type CaptchaRepository interface {
	StoreCaptchaAnswer(ctx context.Context, captchaID string, x, y int, expiry time.Duration) error
	GetCaptchaAnswer(ctx context.Context, captchaID string) (int, int, error)
	DeleteCaptchaAnswer(ctx context.Context, captchaID string) error
	StoreClickAnswer(ctx context.Context, captchaID string, dots map[int]*click.Dot, expiry time.Duration) error
	GetClickAnswer(ctx context.Context, captchaID string) (map[int]*click.Dot, error)
	DeleteClickAnswer(ctx context.Context, captchaID string) error
	StoreRotateAnswer(ctx context.Context, captchaID string, angle int, expiry time.Duration) error
	GetRotateAnswer(ctx context.Context, captchaID string) (int, error)
	DeleteRotateAnswer(ctx context.Context, captchaID string) error
	StoreCaptchaToken(ctx context.Context, token string, expiry time.Duration) error
	ValidateAndDeleteCaptchaToken(ctx context.Context, token string) error
}

type CaptchaService struct {
	Slide       *captcha.SlideCaptcha
	Click       *captcha.ClickCaptcha
	Rotate      *captcha.RotateCaptcha
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

func (s *CaptchaService) ValidateSlideCaptcha(ctx context.Context, captchaID string, x, y int) (string, error) {
	targetX, targetY, err := s.CaptchaRepo.GetCaptchaAnswer(ctx, captchaID)
	if err != nil {
		s.Logger.Ctx(ctx).Warn("get captcha answer failed", zap.String("captchaID", captchaID), zap.Error(err))
		return "", response.CaptchaExpired
	}

	if err := s.CaptchaRepo.DeleteCaptchaAnswer(ctx, captchaID); err != nil {
		s.Logger.Ctx(ctx).Error("delete captcha answer failed", zap.String("captchaID", captchaID), zap.Error(err))
	}

	if !captcha.ValidateSlide(x, y, targetX, targetY, 5) {
		return "", response.CaptchaInvalid
	}

	return s.issueToken(ctx)
}

func (s *CaptchaService) GenerateClickCaptcha(ctx context.Context) (*resp.ClickCaptchaResponse, error) {
	data, err := s.Click.Generate()
	if err != nil {
		s.Logger.Ctx(ctx).Error("generate click captcha failed", zap.Error(err))
		return nil, response.CaptchaGenFailed
	}

	captchaID := uuid.New().String()
	if err := s.CaptchaRepo.StoreClickAnswer(ctx, captchaID, data.Dots, captchaExpiry); err != nil {
		s.Logger.Ctx(ctx).Error("store click captcha answer failed", zap.Error(err))
		return nil, response.CaptchaGenFailed
	}

	return &resp.ClickCaptchaResponse{
		CaptchaID:   captchaID,
		MasterImage: data.MasterImage,
		ThumbImage:  data.ThumbImage,
	}, nil
}

func (s *CaptchaService) ValidateClickCaptcha(ctx context.Context, captchaID string, dots []struct{ X, Y int }) (string, error) {
	targetDots, err := s.CaptchaRepo.GetClickAnswer(ctx, captchaID)
	if err != nil {
		s.Logger.Ctx(ctx).Warn("get click captcha answer failed", zap.String("captchaID", captchaID), zap.Error(err))
		return "", response.CaptchaExpired
	}

	if err := s.CaptchaRepo.DeleteClickAnswer(ctx, captchaID); err != nil {
		s.Logger.Ctx(ctx).Error("delete click captcha answer failed", zap.String("captchaID", captchaID), zap.Error(err))
	}

	if len(dots) != len(targetDots) {
		return "", response.CaptchaInvalid
	}

	for i, dot := range dots {
		target, ok := targetDots[i]
		if !ok {
			return "", response.CaptchaInvalid
		}
		if !captcha.ValidateClick(dot.X, dot.Y, target.X, target.Y, target.Width, target.Height, 15) {
			return "", response.CaptchaInvalid
		}
	}

	return s.issueToken(ctx)
}

func (s *CaptchaService) GenerateRotateCaptcha(ctx context.Context) (*resp.RotateCaptchaResponse, error) {
	data, err := s.Rotate.Generate()
	if err != nil {
		s.Logger.Ctx(ctx).Error("generate rotate captcha failed", zap.Error(err))
		return nil, response.CaptchaGenFailed
	}

	captchaID := uuid.New().String()
	if err := s.CaptchaRepo.StoreRotateAnswer(ctx, captchaID, data.Angle, captchaExpiry); err != nil {
		s.Logger.Ctx(ctx).Error("store rotate captcha answer failed", zap.Error(err))
		return nil, response.CaptchaGenFailed
	}

	return &resp.RotateCaptchaResponse{
		CaptchaID:   captchaID,
		MasterImage: data.MasterImage,
		ThumbImage:  data.ThumbImage,
	}, nil
}

func (s *CaptchaService) ValidateRotateCaptcha(ctx context.Context, captchaID string, angle int) (string, error) {
	targetAngle, err := s.CaptchaRepo.GetRotateAnswer(ctx, captchaID)
	if err != nil {
		s.Logger.Ctx(ctx).Warn("get rotate captcha answer failed", zap.String("captchaID", captchaID), zap.Error(err))
		return "", response.CaptchaExpired
	}

	if err := s.CaptchaRepo.DeleteRotateAnswer(ctx, captchaID); err != nil {
		s.Logger.Ctx(ctx).Error("delete rotate captcha answer failed", zap.String("captchaID", captchaID), zap.Error(err))
	}

	if !captcha.ValidateRotate(angle, targetAngle, 5) {
		return "", response.CaptchaInvalid
	}

	return s.issueToken(ctx)
}

func (s *CaptchaService) ValidateCaptchaToken(ctx context.Context, token string) error {
	if err := s.CaptchaRepo.ValidateAndDeleteCaptchaToken(ctx, token); err != nil {
		return response.CaptchaTokenInvalid
	}
	return nil
}

func (s *CaptchaService) issueToken(ctx context.Context) (string, error) {
	token := uuid.New().String()
	if err := s.CaptchaRepo.StoreCaptchaToken(ctx, token, tokenExpiry); err != nil {
		s.Logger.Ctx(ctx).Error("store captcha token failed", zap.Error(err))
		return "", response.InternalError
	}
	return token, nil
}
