package main

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

func listEchosHandler(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	if page <= 0 {
		return errors.New("invalid page")
	}

	echos, err := database.FindAll((page-1)*15, 15)
	if err != nil {
		return err
	}

	c.JSON(echos)

	return nil
}

func deleteEchoHandler(c *fiber.Ctx) error {
	hash := c.Query("hash")
	if hash == "" {
		return errors.New("invalid hash")
	}

	echo, err := database.Find(hash)
	if err != nil {
		return err
	}

	if echo == nil {
		return errors.New("echo not found")
	}

	_ = echo.Unlink()
	_ = database.Delete(hash)

	c.JSON(map[string]interface{}{
		"success": true,
	})

	return nil
}
