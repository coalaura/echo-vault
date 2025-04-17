package main

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"os"

	"github.com/gen2brain/webp"
)

func (e *Echo) SaveUploadedFile(header *multipart.FileHeader, lossless bool) error {
	file, err := header.Open()
	if err != nil {
		return err
	}

	defer file.Close()

	err = e.Fill()
	if err != nil {
		return err
	}

	switch e.Extension {
	case "jpg", "jpeg", "png", "webp":
		e.Extension = "webp"

		return saveImageAsWebP(file, e.Storage(), lossless)
	case "gif":
		return saveFileAsFile(file, e.Storage())
	}

	return nil
}

// PNG, JPG -> WebP
func saveImageAsWebP(file multipart.File, path string, lossless bool) error {
	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	defer out.Close()

	return webp.Encode(out, img, getWebPOptions(lossless))
}

// ANY -> ANY
func saveFileAsFile(file multipart.File, path string) error {
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	defer out.Close()

	_, err = io.Copy(out, file)

	return err
}

// For the scanner
func convertEchoToWebP(echo *Echo) error {
	source := echo.Storage()

	file, err := os.Open(source)
	if err != nil {
		return err
	}

	echo.Extension = "webp"

	err = saveImageAsWebP(file, echo.Storage(), false)
	if err != nil {
		_ = file.Close()

		return err
	}

	_ = file.Close()
	_ = os.Remove(source)

	return nil
}

func getWebPOptions(lossless bool) webp.Options {
	opts := webp.Options{
		Method: 6, // Max
	}

	if lossless || config.Quality <= 0 || config.Quality >= 100 {
		opts.Lossless = true
		opts.Exact = true
	} else {
		opts.Quality = int(config.Quality)
	}

	return opts
}
