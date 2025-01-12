package main

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func authenticate(c *fiber.Ctx) error {
	token := c.Get("Authorization")

	if !strings.HasPrefix(token, "Bearer ") {
		return errors.New("invalid token")
	}

	token = strings.TrimPrefix(token, "Bearer ")

	if token != config.UploadToken || token == "" {
		return errors.New("invalid token")
	}

	c.Next()

	return nil
}
