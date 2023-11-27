package main

import (
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Echo struct {
	ID         int64
	Hash       string
	Name       string
	Extension  string
	UploadSize int64
	Timestamp  int64
}

func uploadHandler(c *gin.Context) {
	token := c.PostForm("token")

	if token != config.UploadToken {
		errorResponse(c, 401, "invalid token")

		return
	}

	echo, header, err := validateUpload(c)
	if err != nil {
		errorResponse(c, 400, err.Error())

		return
	}

	err = echo.Create()
	if err != nil {
		errorResponse(c, 500, err.Error())

		return
	}

	err = c.SaveUploadedFile(header, echo.Storage())
	if err != nil {
		_ = echo.Delete()

		errorResponse(c, 500, err.Error())

		return
	}

	echo.Compress()

	uploadResponse(c, echo)
}

func validateUpload(c *gin.Context) (*Echo, *multipart.FileHeader, error) {
	header, err := c.FormFile("file")
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
