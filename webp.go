package main

import (
	"errors"
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"os"

	"image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/gen2brain/webp"
)

var ErrGifNoEncode = errors.New("gif is already in final format")

func (e *Echo) SaveUploadedFile(header *multipart.FileHeader) (int, error) {
	file, err := header.Open()
	if err != nil {
		return 0, err
	}

	defer file.Close()

	err = e.Fill()
	if err != nil {
		return 0, err
	}

	switch e.Extension {
	case "jpg", "jpeg", "png", "webp":
		e.Extension = "webp"

		return saveImageAsWebP(file, e.Storage())
	case "gif":
		if config.Settings.ReEncodeGif {
			return saveGifAsGif(file, e.Storage())
		}

		return saveFileAsFile(file, e.Storage())
	}

	return 0, fmt.Errorf("unsupported extension %q", e.Extension)
}

// PNG, JPG -> WebP
func saveImageAsWebP(file multipart.File, path string) (int, error) {
	img, _, err := image.Decode(file)
	if err != nil {
		return 0, err
	}

	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return 0, err
	}

	defer out.Close()

	wr := NewCountWriter(out)

	err = webp.Encode(wr, img, getWebPOptions())

	return wr.N, err
}

// GIF -> GIF (re-encode, remove metadata)
func saveGifAsGif(file multipart.File, path string) (int, error) {
	img, err := gif.DecodeAll(file)
	if err != nil {
		return 0, err
	}

	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return 0, err
	}

	defer out.Close()

	wr := NewCountWriter(out)

	err = gif.EncodeAll(wr, img)

	return wr.N, err
}

// ANY -> ANY
func saveFileAsFile(file multipart.File, path string) (int, error) {
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return 0, err
	}

	defer out.Close()

	wr := NewCountWriter(out)

	_, err = io.Copy(wr, file)

	return wr.N, err
}

// For the scanner
func convertEchoToWebP(echo *Echo) (bool, error) {
	source := echo.Storage()

	file, err := os.OpenFile(source, os.O_RDONLY, 0)
	if err != nil {
		return false, err
	}

	defer file.Close()

	handler := saveImageAsWebP

	if echo.Extension == "gif" {
		if !config.Settings.ReEncodeGif {
			return false, nil
		}

		handler = saveGifAsGif
	}

	echo.Extension = "webp"

	_, err = handler(file, echo.Storage())
	if err != nil {
		return false, err
	}

	defer os.Remove(source)

	return true, nil
}

func getWebPOptions() webp.Options {
	opts := webp.Options{
		Method: int(config.Settings.Effort),
	}

	if config.Settings.Quality <= 0 || config.Settings.Quality >= 100 {
		opts.Lossless = true
	} else {
		opts.Quality = int(config.Settings.Quality)
	}

	return opts
}
