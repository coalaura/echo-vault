package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
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

	anim, err := webp.DecodeAll(bytes.NewReader(data), getWebPDecodeOptions())
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

	if len(gifImg.Image) == 0 {
		return 0, errNoFrames
	}

	background := color.RGBA{0, 0, 0, 0}

	if gifImg.Config.ColorModel != nil {
		if p, ok := gifImg.Config.ColorModel.(color.Palette); ok && int(gifImg.BackgroundIndex) < len(p) {
			background = color.RGBAModel.Convert(p[gifImg.BackgroundIndex]).(color.RGBA)
		}
	}

	bounds := image.Rect(0, 0, gifImg.Config.Width, gifImg.Config.Height)
	canvas := image.NewRGBA(bounds)

	draw.Draw(canvas, bounds, &image.Uniform{background}, image.Point{}, draw.Src)

	frames := make([]image.Image, len(gifImg.Image))

	var prevCanvas *image.RGBA

	for i, srcFrame := range gifImg.Image {
		if i < len(gifImg.Disposal) && gifImg.Disposal[i] == 3 {
			prevCanvas = image.NewRGBA(bounds)

			draw.Draw(prevCanvas, bounds, canvas, bounds.Min, draw.Src)
		}

		draw.Draw(canvas, srcFrame.Bounds(), srcFrame, srcFrame.Bounds().Min, draw.Over)

		frameCopy := image.NewRGBA(bounds)

		draw.Draw(frameCopy, bounds, canvas, bounds.Min, draw.Src)

		frames[i] = frameCopy

		if i < len(gifImg.Image)-1 {
			disposal := byte(0)

			if i < len(gifImg.Disposal) {
				disposal = gifImg.Disposal[i]
			}

			switch disposal {
			case 0, 1: // No disposal specified or do not dispose - keep canvas as is
				// Do nothing
			case 2: // Restore to background color - clear frame area to background
				draw.Draw(canvas, srcFrame.Bounds(), &image.Uniform{background}, image.Point{}, draw.Src)
			case 3: // Restore to previous - restore canvas to state before this frame
				if prevCanvas != nil {
					draw.Draw(canvas, bounds, prevCanvas, bounds.Min, draw.Src)
				}
			}
		}
	}

	anim := &webp.Animation{
		Image:      frames,
		Delay:      make([]int, len(gifImg.Delay)),
		LoopCount:  gifImg.LoopCount,
		Background: background,
	}

	for i, delay := range gifImg.Delay {
		if delay < 0 {
			delay = 0
		}

		anim.Delay[i] = delay * 10
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

	anim, err := webp.DecodeAll(bytes.NewReader(data), getWebPDecodeOptions())
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

func getWebPDecodeOptions() *webp.DecodeOptions {
	return &webp.DecodeOptions{
		UseThreads: true,
	}
}
