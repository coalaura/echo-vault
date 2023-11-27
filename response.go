package main

import "github.com/gin-gonic/gin"

func succeed(c *gin.Context, data interface{}) {
	c.JSON(200, data)
}

func fail(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, gin.H{
		"error": message,
	})
}
