package v1

import (
	"context"

	"github.com/samber/lo"
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
	GetPermissionsForRoles(ctx context.Context, roleIDs []uint64) ([]string, []string, error)
	GetModulePermissions(ctx context.Context, roleIDs []uint64, module string) ([]string, error)
	GetUserWithRoles(ctx context.Context, userID uint64) (*model.User, error)
	VerifyEmail(ctx context.Context, email, code string) error
	ResendVerifyCode(ctx context.Context, email string) error
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
		auth.POST("/verify-email", h.VerifyEmail)
		auth.POST("/resend-verify-code", h.ResendVerifyCode)
	}
	authRequired := r.Group("/auth")
	authRequired.Use(h.JWTAuth)
	{
		authRequired.POST("/logout", h.Logout)
		authRequired.GET("/permissions", h.GetPermissions)
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
	roleIDs := make([]uint64, 0, len(user.Roles))
	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roleIDs = append(roleIDs, role.ID)
		roles = append(roles, role.Code)
	}
	menuPerms, routePerms, err := h.Service.GetPermissionsForRoles(c.Request.Context(), roleIDs)
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
		Token:            pair.AccessToken,
		RefreshToken:     pair.RefreshToken,
		Role:             roles,
		MenuPermissions:  menuPerms,
		RoutePermissions: routePerms,
		CompletedTours:   lo.Ternary(user.CompletedTours == nil, []string{}, user.CompletedTours),
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

// GetPermissions
// @Summary      获取模块权限
// @Description  返回当前用户在指定模块下的权限码列表
// @Tags         认证
// @Produce      json
// @Security     BearerAuth
// @Param        module  query  string  true  "模块名称"
// @Success      200  {object}  response.BasicResponse[resp.PermissionsResponse]
// @Router       /api/v1/auth/permissions [get]
func (h *AuthHandler) GetPermissions(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.FailWithCode(c, response.InvalidToken)
		return
	}
	module := c.Query("module")
	if module == "" {
		response.FailWithCode(c, response.InvalidParams)
		return
	}
	user, err := h.Service.GetUserWithRoles(c.Request.Context(), userID)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	roleIDs := make([]uint64, 0, len(user.Roles))
	for _, r := range user.Roles {
		roleIDs = append(roleIDs, r.ID)
	}
	perms, err := h.Service.GetModulePermissions(c.Request.Context(), roleIDs, module)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.SuccessWithData(c, resp.PermissionsResponse{
		Permissions: perms,
	})
}

// VerifyEmail
// @Summary      邮箱验证
// @Description  使用验证码完成邮箱验证，激活账户
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body  req.VerifyEmailRequest  true  "验证信息"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /api/v1/auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var r req.VerifyEmailRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	if err := h.Service.VerifyEmail(c.Request.Context(), r.Email, r.Code); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c)
}

// ResendVerifyCode
// @Summary      重新发送验证码
// @Description  重新发送邮箱验证码，需要验证人机验证码
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body  req.ResendVerifyCodeRequest  true  "请求信息"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Router       /api/v1/auth/resend-verify-code [post]
func (h *AuthHandler) ResendVerifyCode(c *gin.Context) {
	var r req.ResendVerifyCodeRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	if err := h.CaptchaService.ValidateCaptchaToken(c.Request.Context(), r.CaptchaToken); err != nil {
		response.FailWithCode(c, response.CaptchaTokenInvalid)
		return
	}
	if err := h.Service.ResendVerifyCode(c.Request.Context(), r.Email); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c)
}
