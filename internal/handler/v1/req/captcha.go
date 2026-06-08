package req

type ClickDot struct {
	X int `json:"x" binding:"required"`
	Y int `json:"y" binding:"required"`
}

type ValidateCaptchaRequest struct {
	CaptchaType string     `json:"captcha_type" binding:"required,oneof=slide click rotate"`
	CaptchaID   string     `json:"captcha_id" binding:"required"`
	X           int        `json:"x"`
	Y           int        `json:"y"`
	Dots        []ClickDot `json:"dots"`
	Angle       int        `json:"angle"`
}
