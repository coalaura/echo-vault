package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"os"

	"git.sr.ht/~jackmordaunt/go-libwebp/webp"
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

	return fmt.Errorf("unsupported extension %q", e.Extension)
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

	return webp.Encode(out, img, getWebPOptions(lossless)...)
}

// ANY -> ANY
func saveFileAsFile(file multipart.File, path string) error {
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
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

	file, err := os.OpenFile(source, os.O_RDONLY, 0)
	if err != nil {
		return err
	}

	defer file.Close()

	echo.Extension = "webp"

	err = saveImageAsWebP(file, echo.Storage(), false)
	if err != nil {
		return err
	}

	defer os.Remove(source)

	return nil
}

func getWebPOptions(lossless bool) []webp.EncodeOption {
	var opts []webp.EncodeOption

	if lossless || config.Quality <= 0 || config.Quality >= 100 {
		opts = append(opts, webp.Lossless())
	} else {
		opts = append(opts, webp.Quality(float32(config.Quality)/100))
	}

	return opts
}
