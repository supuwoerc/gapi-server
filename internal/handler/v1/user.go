package v1

import (
	"context"

	"github.com/samber/lo"
	"github.com/supuwoerc/gapi-server/internal/dal/model"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/req"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/resp"
	"github.com/supuwoerc/gapi-server/internal/middleware"
	"github.com/supuwoerc/gapi-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserServiceInterface interface {
	GetProfile(ctx context.Context, userID uint64) (*model.User, error)
	GetMenuPermissions(ctx context.Context, roleIDs []uint64) ([]string, error)
	GetRoutePermissions(ctx context.Context, roleIDs []uint64) ([]string, error)
	UpdateProfile(ctx context.Context, userID uint64, name, bio, avatar string) (*model.User, error)
	UpdateCompletedTours(ctx context.Context, userID uint64, tours []string) ([]string, error)
}

type UserHandler struct {
	Service UserServiceInterface
	JWTAuth gin.HandlerFunc
}

func (h *UserHandler) Register(r *gin.RouterGroup) {
	user := r.Group("/user")
	user.Use(h.JWTAuth)
	{
		user.GET("/profile", h.GetProfile)
		user.PATCH("/profile", h.UpdateProfile)
		user.PATCH("/tour", h.UpdateTour)
	}
}

// GetProfile
// @Summary      获取当前用户信息
// @Description  返回当前登录用户的基本信息、角色、权限和已完成引导
// @Tags         用户
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.BasicResponse[resp.LoginResponse]
// @Failure      401  {object}  response.Response
// @Router       /api/v1/user/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.FailWithCode(c, response.InvalidToken)
		return
	}
	user, err := h.Service.GetProfile(c.Request.Context(), userID)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	roleIDs := make([]uint64, 0, len(user.Roles))
	roles := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roleIDs = append(roleIDs, r.ID)
		roles = append(roles, r.Code)
	}
	menuPerms, err := h.Service.GetMenuPermissions(c.Request.Context(), roleIDs)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	routePerms, err := h.Service.GetRoutePermissions(c.Request.Context(), roleIDs)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.SuccessWithData(c, resp.LoginResponse{
		User: resp.UserInfo{
			Name:   user.Username,
			Email:  user.Email,
			Avatar: user.Avatar,
			Bio:    user.Bio,
		},
		Role:             roles,
		MenuPermissions:  menuPerms,
		RoutePermissions: routePerms,
		CompletedTours:   lo.Ternary(user.CompletedTours == nil, []string{}, user.CompletedTours),
	})
}

// UpdateProfile
// @Summary      更新用户资料
// @Description  更新当前用户的名称、简介和头像
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  req.UpdateProfileRequest  true  "资料信息"
// @Success      200  {object}  response.BasicResponse[resp.UpdateProfileResponse]
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /api/v1/user/profile [patch]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
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

// UpdateTour
// @Summary      更新已完成引导
// @Description  追加当前用户已完成的引导 ID 列表
// @Tags         用户
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  req.UpdateToursRequest  true  "引导信息"
// @Success      200  {object}  response.BasicResponse[resp.UpdateToursResponse]
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /api/v1/user/tour [patch]
func (h *UserHandler) UpdateTour(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.FailWithCode(c, response.InvalidToken)
		return
	}
	var r req.UpdateToursRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	tours, err := h.Service.UpdateCompletedTours(c.Request.Context(), userID, r.CompletedTours)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.SuccessWithData(c, resp.UpdateToursResponse{
		CompletedTours: tours,
	})
}
