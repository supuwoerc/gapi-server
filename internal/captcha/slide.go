package captcha

import (
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/wenlng/go-captcha/v2/slide"
)

type SlideData struct {
	MasterImage string
	TileImage   string
	X           int
	Y           int
}

type SlideCaptcha struct {
	captcha slide.Captcha
}

func NewSlideCaptcha() (*SlideCaptcha, error) {
	builder := slide.NewBuilder()

	backgrounds := []image.Image{generateBackground(300, 220)}
	graphs := []*slide.GraphImage{generateGraphImage(60, 60)}

	builder.SetResources(
		slide.WithBackgrounds(backgrounds),
		slide.WithGraphImages(graphs),
	)

	capt := builder.Make()
	return &SlideCaptcha{captcha: capt}, nil
}

func NewSlideCaptchaWithResources(backgrounds []image.Image, graphs []*slide.GraphImage) (*SlideCaptcha, error) {
	builder := slide.NewBuilder()
	builder.SetResources(
		slide.WithBackgrounds(backgrounds),
		slide.WithGraphImages(graphs),
	)
	capt := builder.Make()
	return &SlideCaptcha{captcha: capt}, nil
}

func (s *SlideCaptcha) Generate() (*SlideData, error) {
	captData, err := s.captcha.Generate()
	if err != nil {
		return nil, err
	}

	block := captData.GetData()

	masterBytes, err := captData.GetMasterImage().ToBytes()
	if err != nil {
		return nil, err
	}
	tileBytes, err := captData.GetTileImage().ToBytes()
	if err != nil {
		return nil, err
	}

	return &SlideData{
		MasterImage: base64.StdEncoding.EncodeToString(masterBytes),
		TileImage:   base64.StdEncoding.EncodeToString(tileBytes),
		X:           block.X,
		Y:           block.Y,
	}, nil
}

func ValidateSlide(srcX, srcY, targetX, targetY, padding int) bool {
	return slide.Validate(srcX, srcY, targetX, targetY, padding)
}

func generateBackground(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			r := uint8((x*7 + y*3) % 200)
			g := uint8((x*3 + y*7 + 50) % 200)
			b := uint8((x*5 + y*5 + 100) % 200)
			img.Set(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func generateGraphImage(w, h int) *slide.GraphImage {
	overlay := image.NewNRGBA(image.Rect(0, 0, w, h))
	shadow := image.NewNRGBA(image.Rect(0, 0, w, h))
	mask := image.NewNRGBA(image.Rect(0, 0, w, h))

	cx, cy := float64(w)/2, float64(h)/2
	radius := float64(w) / 2.2

	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			dist := math.Sqrt(math.Pow(float64(x)-cx, 2) + math.Pow(float64(y)-cy, 2))
			if dist <= radius {
				overlay.Set(x, y, color.NRGBA{R: 180, G: 180, B: 180, A: 255})
				shadow.Set(x, y, color.NRGBA{R: 0, G: 0, B: 0, A: 120})
				mask.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
			}
		}
	}

	draw.Draw(overlay, overlay.Bounds(), overlay, image.Point{}, draw.Src)

	return &slide.GraphImage{
		OverlayImage: overlay,
		ShadowImage:  shadow,
		MaskImage:    mask,
	}
}
