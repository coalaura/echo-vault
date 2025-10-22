package main

import (
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/gen2brain/webp"
)

func decodeImage(rd io.Reader) (image.Image, error) {
	img, _, err := image.Decode(rd)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func saveImageAsWebP(rd io.Reader, path string) (int64, error) {
	img, err := decodeImage(rd)
	if err != nil {
		return 0, err
	}

	wr, err := OpenCountWriter(path)
	if err != nil {
		return 0, err
	}

	defer wr.Close()

	err = webp.Encode(wr, img, getWebPOptions())

	return wr.N, err
}

func saveImageAsPNG(rd io.Reader, path string) (int64, error) {
	img, err := decodeImage(rd)
	if err != nil {
		return 0, err
	}

	wr, err := OpenCountWriter(path)
	if err != nil {
		return 0, err
	}

	defer wr.Close()

	err = getPNGEncoder().Encode(wr, img)

	return wr.N, err
}

func saveImageAsJPEG(rd io.Reader, path string) (int64, error) {
	img, err := decodeImage(rd)
	if err != nil {
		return 0, err
	}

	wr, err := OpenCountWriter(path)
	if err != nil {
		return 0, err
	}

	defer wr.Close()

	err = jpeg.Encode(wr, img, getJPEGOptions())

	return wr.N, err
}

func getWebPOptions() webp.Options {
	var opts webp.Options

	switch config.Images.Effort {
	case 1:
		opts.Method = 1
	case 2:
		opts.Method = webp.DefaultMethod
	case 3:
		opts.Method = 6
	}

	if config.Images.Quality == 100 {
		opts.Lossless = true
	} else {
		opts.Quality = int(config.Images.Quality)
	}

	return opts
}

func getPNGEncoder() *png.Encoder {
	var enc png.Encoder

	switch config.Images.Effort {
	case 1:
		enc.CompressionLevel = png.BestSpeed
	case 2:
		enc.CompressionLevel = png.DefaultCompression
	case 3:
		enc.CompressionLevel = png.BestCompression
	}

	return &enc
}

func getJPEGOptions() *jpeg.Options {
	var opts jpeg.Options

	opts.Quality = config.Images.Quality

	return &opts
}
