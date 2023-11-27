package main

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func authenticate(c *gin.Context) {
	token := c.GetHeader("Authorization")

	if !strings.HasPrefix(token, "Bearer ") {
		fail(c, 401, "missing token")

		return
	}

	token = strings.TrimPrefix(token, "Bearer ")

	if token != config.UploadToken || token == "" {
		fail(c, 401, "invalid token")

		return
	}

	c.Next()
}
