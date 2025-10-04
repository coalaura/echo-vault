package main

import (
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func uploadHandler(c *fiber.Ctx) error {
	log.Printf("Received upload request from %s\n", c.IP())

	echo, header, err := validateUpload(c)
	if err != nil {
		log.Warnf("Failed to validate upload: %v\n", err)

		return err
	}

	err = echo.SaveUploadedFile(header, c.QueryBool("lossless"))
	if err != nil {
		log.Warnf("Failed to save uploaded file: %v\n", err)

		return err
	}

	err = database.Create(echo)
	if err != nil {
		log.Warnf("Failed to create echo in database: %v\n", err)

		return err
	}

	c.JSON(map[string]interface{}{
		"hash":      echo.Hash,
		"extension": echo.Extension,
		"url":       echo.URL(),
	})

	return nil
}

func validateUpload(c *fiber.Ctx) (*Echo, *multipart.FileHeader, error) {
	header, err := c.FormFile("upload")
	if err != nil {
		return nil, nil, err
	}

	if header.Size > config.MaxFileSize() {
		return nil, nil, fmt.Errorf("file too large")
	}

	file, err := header.Open()
	if err != nil {
		return nil, nil, err
	}

	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return nil, nil, err
	}

	echo := Echo{
		Name:       header.Filename,
		UploadSize: header.Size,
	}

	contentType := http.DetectContentType(buffer)

	switch contentType {
	case "image/jpeg":
		echo.Extension = "jpg"
	case "image/png":
		echo.Extension = "png"
	case "image/gif":
		echo.Extension = "gif"
	case "image/webp":
		echo.Extension = "webp"
	default:
		return nil, nil, fmt.Errorf("invalid file type %q", contentType)
	}

	return &echo, header, nil
}
