package provider

import (
	"github.com/supuwoerc/gapi-server/internal/captcha"
	"github.com/supuwoerc/gapi-server/internal/dal"
	v1 "github.com/supuwoerc/gapi-server/internal/handler/v1"
	"github.com/supuwoerc/gapi-server/internal/service"

	"github.com/google/wire"
)

var CaptchaSet = wire.NewSet(
	ProvideSlideCaptcha,
	wire.Struct(new(dal.CaptchaDal), "*"),
	wire.Struct(new(service.CaptchaService), "*"),
	wire.Bind(new(service.CaptchaRepository), new(*dal.CaptchaDal)),
	wire.Bind(new(v1.CaptchaServiceInterface), new(*service.CaptchaService)),
	wire.Struct(new(v1.CaptchaHandler), "*"),
)

func ProvideSlideCaptcha() *captcha.SlideCaptcha {
	sc, err := captcha.NewSlideCaptcha()
	if err != nil {
		panic("failed to initialize slide captcha: " + err.Error())
	}
	return sc
}
