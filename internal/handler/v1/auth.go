package v1

import (
	"context"

	"github.com/supuwoerc/gapi-server/internal/dal/model"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/req"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/resp"
	"github.com/supuwoerc/gapi-server/internal/middleware"
	"github.com/supuwoerc/gapi-server/pkg/jwt"
	"github.com/supuwoerc/gapi-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuthServiceInterface interface {
	Register(ctx context.Context, username, email, password string) error
	Login(ctx context.Context, email, password string) (*jwt.TokenPair, *model.User, error)
	RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error)
	Logout(ctx context.Context, userID uint64)
	GetProfile(ctx context.Context, userID uint64) (*model.User, error)
}

type AuthHandler struct {
	Service        AuthServiceInterface
	CaptchaService CaptchaServiceInterface
	JWTAuth        gin.HandlerFunc
}

// Register registers auth routes on the given router group.
func (h *AuthHandler) Register(r *gin.RouterGroup) {
	auth := r.Group("/auth")
	{
		auth.POST("/sign-up", h.SignUp)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
	}
	authRequired := r.Group("/auth")
	authRequired.Use(h.JWTAuth)
	{
		authRequired.POST("/logout", h.Logout)
		authRequired.GET("/profile", h.Profile)
	}
}

// SignUp
// @Summary      用户注册
// @Description  使用用户名、邮箱和密码注册新用户
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body  req.RegisterRequest  true  "注册信息"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /api/v1/auth/sign-up [post]
func (h *AuthHandler) SignUp(c *gin.Context) {
	var r req.RegisterRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	if err := h.CaptchaService.ValidateCaptchaToken(c.Request.Context(), r.CaptchaToken); err != nil {
		response.FailWithCode(c, response.CaptchaTokenInvalid)
		return
	}
	if err := h.Service.Register(c.Request.Context(), r.Username, r.Email, r.Password); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c)
}

// Login
// @Summary      用户登录
// @Description  使用邮箱和密码登录，返回 token 对和用户信息
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body  req.LoginRequest  true  "登录信息"
// @Success      200  {object}  response.BasicResponse[resp.LoginResponse]
// @Failure      400  {object}  response.Response
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var r req.LoginRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	pair, user, err := h.Service.Login(c.Request.Context(), r.Email, r.Password)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.SuccessWithData(c, resp.LoginResponse{
		User: resp.UserInfo{
			Name:   user.Username,
			Email:  user.Email,
			Avatar: user.Avatar,
			Bio:    "",
		},
		Token:            pair.AccessToken,
		RefreshToken:     pair.RefreshToken,
		Role:             []string{},
		MenuPermissions:  []string{},
		RoutePermissions: []string{},
		CompletedTours:   []string{},
	})
}

// Refresh
// @Summary      刷新 Token
// @Description  使用 refresh_token 换取新的 token 对
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body  req.RefreshTokenRequest  true  "刷新请求"
// @Success      200  {object}  response.BasicResponse[resp.RefreshTokenResponse]
// @Failure      400  {object}  response.Response
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var r req.RefreshTokenRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	pair, err := h.Service.RefreshToken(c.Request.Context(), r.RefreshToken)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.SuccessWithData(c, resp.RefreshTokenResponse{
		Token:        pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	})
}

// Logout
// @Summary      用户登出
// @Description  使当前用户的 refresh_token 失效
// @Tags         认证
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Response
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.FailWithCode(c, response.InvalidToken)
		return
	}
	h.Service.Logout(c.Request.Context(), userID)
	response.Success(c)
}

// Profile
// @Summary      获取当前用户信息
// @Description  返回当前登录用户的信息
// @Tags         认证
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.BasicResponse[resp.LoginResponse]
// @Router       /api/v1/auth/profile [get]
func (h *AuthHandler) Profile(c *gin.Context) {
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
	roles := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roles = append(roles, r.Code)
	}
	response.SuccessWithData(c, resp.LoginResponse{
		User: resp.UserInfo{
			Name:   user.Username,
			Email:  user.Email,
			Avatar: user.Avatar,
			Bio:    "",
		},
		Role:             roles,
		MenuPermissions:  []string{},
		RoutePermissions: []string{},
		CompletedTours:   []string{},
	})
}
