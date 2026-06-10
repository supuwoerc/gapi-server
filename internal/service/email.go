package service

import (
	"context"
	"fmt"

	"github.com/supuwoerc/gapi-server/pkg/email"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"go.uber.org/zap"
)

type EmailService struct {
	Sender   email.Sender
	Template *email.TemplateRenderer
	Logger   *logger.Logger
}

type VerificationCodeData struct {
	AppName       string
	Code          string
	ExpireMinutes int
}

func (s *EmailService) SendVerificationCode(ctx context.Context, to string, code string, expireMinutes int) error {
	html, err := s.Template.Render("verification_code.html", VerificationCodeData{
		AppName:       "GAPI",
		Code:          code,
		ExpireMinutes: expireMinutes,
	})
	if err != nil {
		s.Logger.Ctx(ctx).Error("failed to render verification code template", zap.Error(err))
		return err
	}
	return s.Sender.Send(ctx, &email.Message{
		To:      []string{to},
		Subject: "验证码",
		HTML:    html,
		Text:    fmt.Sprintf("您的验证码是：%s，%d 分钟内有效。", code, expireMinutes),
	})
}
