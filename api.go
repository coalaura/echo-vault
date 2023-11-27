package main

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func listEchosHandler(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		fail(c, 400, "invalid page")

		return
	}

	echos, err := database.FindAll((page-1)*15, 15)
	if err != nil {
		fail(c, 500, err.Error())

		return
	}

	c.JSON(200, echos)
}

func deleteEchoHandler(c *gin.Context) {
	hash := c.Param("hash")

	if hash == "" {
		fail(c, 400, "missing hash")

		return
	}

	echo, err := database.Find(hash)
	if err != nil {
		fail(c, 500, err.Error())

		return
	}

	if echo == nil {
		fail(c, 404, "echo not found")

		return
	}

	_ = echo.Unlink()
	_ = database.Delete(hash)

	c.JSON(200, gin.H{"success": true})
}
