package req

type UpdateProfileRequest struct {
	Name   string `json:"name" binding:"required,min=1,max=64"`
	Bio    string `json:"bio" binding:"required,max=256"`
	Avatar string `json:"avatar" binding:"omitempty,max=512"`
}
