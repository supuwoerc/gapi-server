package resp

type CaptchaResponse struct {
	CaptchaID   string `json:"captcha_id"`
	MasterImage string `json:"master_image"`
	TileImage   string `json:"tile_image"`
	TileY       int    `json:"tile_y"`
}
