package main

import "github.com/gin-gonic/gin"

func errorResponse(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, gin.H{
		"error": message,
	})
}

func uploadResponse(c *gin.Context, echo *Echo) {
	c.JSON(200, gin.H{
		"hash":      echo.Hash,
		"extension": echo.Extension,
		"url":       echo.URL(),
	})
}
