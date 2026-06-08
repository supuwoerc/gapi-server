package v1

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/req"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/resp"
	"github.com/supuwoerc/gapi-server/pkg/response"
)

type CaptchaServiceInterface interface {
	GenerateSlideCaptcha(ctx context.Context) (*resp.CaptchaResponse, error)
	ValidateSlideCaptcha(ctx context.Context, captchaID string, x, y int) (string, error)
	GenerateClickCaptcha(ctx context.Context) (*resp.ClickCaptchaResponse, error)
	ValidateClickCaptcha(ctx context.Context, captchaID string, dots []struct{ X, Y int }) (string, error)
	GenerateRotateCaptcha(ctx context.Context) (*resp.RotateCaptchaResponse, error)
	ValidateRotateCaptcha(ctx context.Context, captchaID string, angle int) (string, error)
	ValidateCaptchaToken(ctx context.Context, token string) error
}

type CaptchaHandler struct {
	Service CaptchaServiceInterface
}

func (h *CaptchaHandler) Register(r *gin.RouterGroup) {
	captcha := r.Group("/captcha")
	{
		captcha.GET("/slide", h.GenerateSlide)
		captcha.GET("/click", h.GenerateClick)
		captcha.GET("/rotate", h.GenerateRotate)
		captcha.POST("/validate", h.Validate)
	}
}

// GenerateSlide
// @Summary      生成滑块验证码
// @Description  生成滑块验证码图片及相关信息
// @Tags         验证码
// @Produce      json
// @Success      200  {object}  response.BasicResponse[resp.CaptchaResponse]
// @Failure      500  {object}  response.Response
// @Router       /api/v1/captcha/slide [get]
func (h *CaptchaHandler) GenerateSlide(c *gin.Context) {
	data, err := h.Service.GenerateSlideCaptcha(c.Request.Context())
	if err != nil {
		response.FailWithCode(c, response.CaptchaGenFailed)
		return
	}
	response.SuccessWithData(c, data)
}

// GenerateClick
// @Summary      生成点选验证码
// @Description  生成点选验证码图片及相关信息
// @Tags         验证码
// @Produce      json
// @Success      200  {object}  response.BasicResponse[resp.ClickCaptchaResponse]
// @Failure      500  {object}  response.Response
// @Router       /api/v1/captcha/click [get]
func (h *CaptchaHandler) GenerateClick(c *gin.Context) {
	data, err := h.Service.GenerateClickCaptcha(c.Request.Context())
	if err != nil {
		response.FailWithCode(c, response.CaptchaGenFailed)
		return
	}
	response.SuccessWithData(c, data)
}

// GenerateRotate
// @Summary      生成旋转验证码
// @Description  生成旋转验证码图片及相关信息
// @Tags         验证码
// @Produce      json
// @Success      200  {object}  response.BasicResponse[resp.RotateCaptchaResponse]
// @Failure      500  {object}  response.Response
// @Router       /api/v1/captcha/rotate [get]
func (h *CaptchaHandler) GenerateRotate(c *gin.Context) {
	data, err := h.Service.GenerateRotateCaptcha(c.Request.Context())
	if err != nil {
		response.FailWithCode(c, response.CaptchaGenFailed)
		return
	}
	response.SuccessWithData(c, data)
}

// Validate
// @Summary      验证验证码
// @Description  验证各类验证码并返回一次性 token
// @Tags         验证码
// @Accept       json
// @Produce      json
// @Param        body  body  req.ValidateCaptchaRequest  true  "验证信息"
// @Success      200  {object}  response.BasicResponse[resp.ValidateCaptchaResponse]
// @Failure      400  {object}  response.Response
// @Router       /api/v1/captcha/validate [post]
func (h *CaptchaHandler) Validate(c *gin.Context) {
	var r req.ValidateCaptchaRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}

	var (
		token string
		err   error
	)

	switch r.CaptchaType {
	case "slide":
		token, err = h.Service.ValidateSlideCaptcha(c.Request.Context(), r.CaptchaID, r.X, r.Y)
	case "click":
		dots := make([]struct{ X, Y int }, len(r.Dots))
		for i, d := range r.Dots {
			dots[i] = struct{ X, Y int }{X: d.X, Y: d.Y}
		}
		token, err = h.Service.ValidateClickCaptcha(c.Request.Context(), r.CaptchaID, dots)
	case "rotate":
		token, err = h.Service.ValidateRotateCaptcha(c.Request.Context(), r.CaptchaID, r.Angle)
	}

	if err != nil {
		response.FailWithError(c, err)
		return
	}

	response.SuccessWithData(c, resp.ValidateCaptchaResponse{CaptchaToken: token})
}
