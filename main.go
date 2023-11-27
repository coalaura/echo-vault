package main

import (
	"database/sql"
	"os"

	"github.com/coalaura/logger"
	"github.com/gin-gonic/gin"
)

var (
	config   EchoConfig
	database *sql.DB

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

	r.Static("/", "./storage")

	r.POST("/upload", uploadHandler)

	log.InfoF("Starting server at %s...\n", config.Addr())
	log.MustPanic(r.Run(config.Addr()))
}
