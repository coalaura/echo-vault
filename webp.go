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

func (e *Echo) SaveUploadedFile(header *multipart.FileHeader) error {
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

		return saveImageAsWebP(file, e.Storage())
	case "gif":
		return saveFileAsFile(file, e.Storage())
	}

	return nil
}

// PNG, JPG -> WebP
func saveImageAsWebP(file multipart.File, path string) error {
	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}

	defer out.Close()

	return webp.Encode(out, img, webp.Options{
		Quality: 90,
		Method:  6,
	})
}

// ANY -> ANY
func saveFileAsFile(file multipart.File, path string) error {
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0755)
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

	err = saveImageAsWebP(file, echo.Storage())
	if err != nil {
		_ = file.Close()

		return err
	}

	_ = file.Close()
	_ = os.Remove(source)

	return nil
}
