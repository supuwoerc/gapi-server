package captcha

import (
	"encoding/base64"

	"github.com/wenlng/go-captcha-assets/resources/imagesv2"
	"github.com/wenlng/go-captcha/v2/rotate"
)

type RotateData struct {
	MasterImage string
	ThumbImage  string
	Angle       int
}

type RotateCaptcha struct {
	captcha rotate.Captcha
}

func NewRotateCaptcha() (*RotateCaptcha, error) {
	builder := rotate.NewBuilder()

	imgs, err := imagesv2.GetImages()
	if err != nil {
		return nil, err
	}

	builder.SetResources(
		rotate.WithImages(imgs),
	)

	capt := builder.Make()
	return &RotateCaptcha{captcha: capt}, nil
}

func (r *RotateCaptcha) Generate() (*RotateData, error) {
	captData, err := r.captcha.Generate()
	if err != nil {
		return nil, err
	}

	block := captData.GetData()

	masterBytes, err := captData.GetMasterImage().ToBytes()
	if err != nil {
		return nil, err
	}
	thumbBytes, err := captData.GetThumbImage().ToBytes()
	if err != nil {
		return nil, err
	}

	return &RotateData{
		MasterImage: base64.StdEncoding.EncodeToString(masterBytes),
		ThumbImage:  base64.StdEncoding.EncodeToString(thumbBytes),
		Angle:       block.Angle,
	}, nil
}

func ValidateRotate(srcAngle, targetAngle, padding int) bool {
	return rotate.Validate(srcAngle, targetAngle, padding)
}
