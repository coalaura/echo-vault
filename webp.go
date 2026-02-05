package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"os"

	"github.com/coalaura/webp"
)

type WebPChunk struct {
	FourCC [4]byte
	Size   uint32
}

var errNoFrames = errors.New("animation contains no frames")

func detectAnimatedWebP(path string) (bool, error) {
	config, err := webp.LoadConfigEx(path)
	if err != nil {
		return false, err
	}

	return config.HasAnimation, nil
}

func saveAnimatedWebPAsWebP(input, path string) (int64, error) {
	data, err := os.ReadFile(input)
	if err != nil {
		return 0, err
	}

	anim, err := webp.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return 0, err
	}

	wr, err := OpenCountWriter(path)
	if err != nil {
		return 0, err
	}
	defer wr.Close()

	err = webp.EncodeAll(wr, anim, getWebPOptions())

	return wr.N, err
}

func saveGIFAsAnimatedWebP(input, path string) (int64, error) {
	gifFile, err := os.Open(input)
	if err != nil {
		return 0, err
	}
	defer gifFile.Close()

	gifImg, err := gif.DecodeAll(gifFile)
	if err != nil {
		return 0, err
	}

	anim := &webp.Animation{
		Image:     make([]image.Image, len(gifImg.Image)),
		Delay:     make([]int, len(gifImg.Delay)),
		LoopCount: gifImg.LoopCount,
		Background: color.RGBA{
			R: gifImg.Config.ColorModel.(color.Palette)[0].(color.RGBA).R,
			G: gifImg.Config.ColorModel.(color.Palette)[0].(color.RGBA).G,
			B: gifImg.Config.ColorModel.(color.Palette)[0].(color.RGBA).B,
			A: 255,
		},
	}

	for i, img := range gifImg.Image {
		anim.Image[i] = img
		anim.Delay[i] = gifImg.Delay[i] * 10 // 1/100s to ms
	}

	wr, err := OpenCountWriter(path)
	if err != nil {
		return 0, err
	}
	defer wr.Close()

	err = webp.EncodeAll(wr, anim, getWebPOptions())

	return wr.N, err
}

func extractAnimatedWebPFirstFrame(input, path, format string) (int64, error) {
	data, err := os.ReadFile(input)
	if err != nil {
		return 0, err
	}

	anim, err := webp.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return 0, err
	}

	if len(anim.Image) == 0 {
		return 0, errNoFrames
	}

	firstFrame := anim.Image[0]

	wr, err := OpenCountWriter(path)
	if err != nil {
		return 0, err
	}
	defer wr.Close()

	switch format {
	case "webp":
		err = webp.Encode(wr, firstFrame, getWebPOptions())
	case "png":
		err = getPNGEncoder().Encode(wr, firstFrame)
	case "jpeg":
		err = jpeg.Encode(wr, firstFrame, getJPEGOptions())
	default:
		return 0, fmt.Errorf("unsupported format: %s", format)
	}

	return wr.N, err
}
