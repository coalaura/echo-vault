package main

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"os"

	"github.com/chai2010/webp"
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
	case "jpg", "png":
		e.Extension = "webp"

		return saveImageAsWebP(file, e.Storage())
	case "gif":
		return saveFileAsFile(file, e.Storage())
	case "webp":
		return saveWebPAsWebP(file, e.Storage())
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

	return webp.Encode(out, img, &webp.Options{
		Quality: 90,
	})
}

// WEBP -> WEBP
func saveWebPAsWebP(file multipart.File, path string) error {
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}

	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		return err
	}

	return nil
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