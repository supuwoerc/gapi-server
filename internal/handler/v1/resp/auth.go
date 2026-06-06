package resp

// UserInfo 登录用户信息
type UserInfo struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Avatar string `json:"avatar"`
	Bio    string `json:"bio"`
}

// LoginResponse 登录响应 — 与前端 loginUserSchema 对齐
type LoginResponse struct {
	User             UserInfo `json:"user"`
	Token            string   `json:"token"`
	RefreshToken     string   `json:"refresh_token"`
	Role             []string `json:"role"`
	MenuPermissions  []string `json:"menu_permissions"`
	RoutePermissions []string `json:"route_permissions"`
	CompletedTours   []string `json:"completed_tours"`
}

// RefreshTokenResponse 刷新 token 响应
type RefreshTokenResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}
