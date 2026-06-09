package resp

type UpdateProfileResponse struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Avatar string `json:"avatar"`
	Bio    string `json:"bio"`
}
