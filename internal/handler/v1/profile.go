package v1

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/supuwoerc/gapi-server/internal/dal/model"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/req"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/resp"
	"github.com/supuwoerc/gapi-server/internal/middleware"
	"github.com/supuwoerc/gapi-server/pkg/response"
)

type ProfileServiceInterface interface {
	UpdateProfile(ctx context.Context, userID uint64, name, bio, avatar string) (*model.User, error)
}

type ProfileHandler struct {
	Service ProfileServiceInterface
	JWTAuth gin.HandlerFunc
}

func (h *ProfileHandler) Register(r *gin.RouterGroup) {
	profile := r.Group("/profile")
	profile.Use(h.JWTAuth)
	{
		profile.PATCH("", h.Update)
	}
}

func (h *ProfileHandler) Update(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.FailWithCode(c, response.InvalidToken)
		return
	}
	var r req.UpdateProfileRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	user, err := h.Service.UpdateProfile(c.Request.Context(), userID, r.Name, r.Bio, r.Avatar)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.SuccessWithData(c, resp.UpdateProfileResponse{
		Name:   user.Username,
		Email:  user.Email,
		Avatar: user.Avatar,
		Bio:    user.Bio,
	})
}
