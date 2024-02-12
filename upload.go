package main

import (
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
)

func uploadHandler(c *gin.Context) {
	echo, header, err := validateUpload(c)
	if err != nil {
		fail(c, 400, err.Error())

		return
	}

	err = echo.SaveUploadedFile(header)
	if err != nil {
		fail(c, 500, err.Error())

		return
	}

	err = database.Create(echo)
	if err != nil {
		fail(c, 500, err.Error())

		return
	}

	succeed(c, gin.H{
		"hash":      echo.Hash,
		"extension": echo.Extension,
		"url":       echo.URL(),
	})
}

func validateUpload(c *gin.Context) (*Echo, *multipart.FileHeader, error) {
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
		return nil, nil, fmt.Errorf("invalid file type '%s'", contentType)
	}

	return &echo, header, nil
}
