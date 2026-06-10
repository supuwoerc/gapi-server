package req

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	Username     string `json:"username" binding:"required,min=3,max=64" example:"john_doe"`
	Email        string `json:"email" binding:"required,email,max=128" example:"john@example.com"`
	Password     string `json:"password" binding:"required,min=8,max=128" example:"password123"`
	CaptchaToken string `json:"captcha_token" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// LoginRequest 用户登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"john@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// RefreshTokenRequest 刷新 token 请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// VerifyEmailRequest 邮箱验证请求
type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email" example:"john@example.com"`
	Code  string `json:"code" binding:"required,len=6" example:"123456"`
}

// ResendVerifyCodeRequest 重新发送验证码请求
type ResendVerifyCodeRequest struct {
	Email        string `json:"email" binding:"required,email" example:"john@example.com"`
	CaptchaToken string `json:"captcha_token" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}
