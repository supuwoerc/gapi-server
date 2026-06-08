package resp

type CaptchaResponse struct {
	CaptchaID   string `json:"captcha_id"`
	MasterImage string `json:"master_image"`
	TileImage   string `json:"tile_image"`
	TileY       int    `json:"tile_y"`
}

type ClickCaptchaResponse struct {
	CaptchaID   string `json:"captcha_id"`
	MasterImage string `json:"master_image"`
	ThumbImage  string `json:"thumb_image"`
}

type RotateCaptchaResponse struct {
	CaptchaID   string `json:"captcha_id"`
	MasterImage string `json:"master_image"`
	ThumbImage  string `json:"thumb_image"`
}

type ValidateCaptchaResponse struct {
	CaptchaToken string `json:"captcha_token"`
}
