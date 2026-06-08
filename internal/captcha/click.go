package captcha

import (
	"encoding/base64"

	"github.com/golang/freetype/truetype"
	"github.com/wenlng/go-captcha-assets/bindata/chars"
	"github.com/wenlng/go-captcha-assets/resources/fonts/fzshengsksjw"
	"github.com/wenlng/go-captcha-assets/resources/imagesv2"
	"github.com/wenlng/go-captcha/v2/click"
)

type ClickData struct {
	MasterImage string
	ThumbImage  string
	Dots        map[int]*click.Dot
}

type ClickCaptcha struct {
	captcha click.Captcha
}

func NewClickCaptcha() (*ClickCaptcha, error) {
	builder := click.NewBuilder()

	fonts, err := fzshengsksjw.GetFont()
	if err != nil {
		return nil, err
	}

	imgs, err := imagesv2.GetImages()
	if err != nil {
		return nil, err
	}

	builder.SetResources(
		click.WithChars(chars.GetChineseChars()),
		click.WithFonts([]*truetype.Font{fonts}),
		click.WithBackgrounds(imgs),
	)

	capt := builder.Make()
	return &ClickCaptcha{captcha: capt}, nil
}

func (c *ClickCaptcha) Generate() (*ClickData, error) {
	captData, err := c.captcha.Generate()
	if err != nil {
		return nil, err
	}

	dots := captData.GetData()

	masterBytes, err := captData.GetMasterImage().ToBytes()
	if err != nil {
		return nil, err
	}
	thumbBytes, err := captData.GetThumbImage().ToBytes()
	if err != nil {
		return nil, err
	}

	return &ClickData{
		MasterImage: base64.StdEncoding.EncodeToString(masterBytes),
		ThumbImage:  base64.StdEncoding.EncodeToString(thumbBytes),
		Dots:        dots,
	}, nil
}

func ValidateClick(srcX, srcY, dotX, dotY, width, height, padding int) bool {
	return click.Validate(srcX, srcY, dotX, dotY, width, height, padding)
}
