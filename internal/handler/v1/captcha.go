package v1

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/resp"
	"github.com/supuwoerc/gapi-server/pkg/response"
)

type CaptchaServiceInterface interface {
	GenerateSlideCaptcha(ctx context.Context) (*resp.CaptchaResponse, error)
	ValidateSlideCaptcha(ctx context.Context, captchaID string, x, y int) (bool, error)
}

type CaptchaHandler struct {
	Service CaptchaServiceInterface
}

func (h *CaptchaHandler) Register(r *gin.RouterGroup) {
	captcha := r.Group("/captcha")
	{
		captcha.GET("/slide", h.GenerateSlide)
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
