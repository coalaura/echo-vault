//go:build ai

package main

import (
	"image"
	"image/draw"
	"os"

	xdraw "golang.org/x/image/draw"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

var (
	clipMean = [3]float32{0.48145466, 0.4578275, 0.40821073}
	clipStd  = [3]float32{0.26862954, 0.26130258, 0.27577711}
)

func PreprocessImageFileToNCHW(path string) ([]float32, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return PreprocessImageToNCHW(img), nil
}

func PreprocessImageToNCHW(src image.Image) []float32 {
	sb := src.Bounds()

	sw := sb.Dx()
	sh := sb.Dy()

	var rw, rh int

	if sw < sh {
		rw = 224
		rh = int(float64(sh) * (224.0 / float64(sw)))
	} else {
		rh = 224
		rw = int(float64(sw) * (224.0 / float64(sh)))
	}

	resized := image.NewRGBA(image.Rect(0, 0, rw, rh))

	xdraw.CatmullRom.Scale(resized, resized.Bounds(), src, sb, draw.Over, nil)

	offX := (rw - 224) / 2
	offY := (rh - 224) / 2

	crop := image.NewRGBA(image.Rect(0, 0, 224, 224))

	draw.Draw(crop, crop.Bounds(), resized, image.Point{X: offX, Y: offY}, draw.Src)

	out := make([]float32, 3*224*224)

	plane := 224 * 224

	for y := 0; y < 224; y++ {
		for x := 0; x < 224; x++ {
			r, g, b, _ := crop.At(x, y).RGBA()

			rf := float32(r) / 65535.0
			gf := float32(g) / 65535.0
			bf := float32(b) / 65535.0

			i := y*224 + x

			out[0*plane+i] = (rf - clipMean[0]) / clipStd[0]
			out[1*plane+i] = (gf - clipMean[1]) / clipStd[1]
			out[2*plane+i] = (bf - clipMean[2]) / clipStd[2]
		}
	}

	return out
}
