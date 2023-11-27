package main

import (
	"os"

	"github.com/coalaura/logger"
	"github.com/gin-gonic/gin"
)

var (
	config   EchoConfig
	database *EchoDatabase

	log = logger.New()
)

func main() {
	_ = os.MkdirAll("./storage", 0755)

	gin.SetMode(gin.ReleaseMode)

	log.Info("Loading env...")
	log.MustPanic(loadConfig())

	log.Info("Connecting to database...")
	log.MustPanic(connectToDatabase())

	handleTasks()

	log.Info("Configuring gin...")

	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(log.Middleware())
	r.Use(authenticate)

	r.POST("/upload", uploadHandler)

	r.GET("/echos", listEchosHandler)
	r.DELETE("/echos/:hash", deleteEchoHandler)

	log.InfoF("Starting server at %s...\n", config.Addr())
	log.MustPanic(r.Run(config.Addr()))
}
