package main

import (
	"os"

	"github.com/coalaura/logger"
	adapter "github.com/coalaura/logger/fiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

var (
	config   EchoConfig
	database *EchoDatabase

	log = logger.New()
)

func main() {
	os.MkdirAll("./storage", 0755)

	log.Info("Loading env...")
	log.MustPanic(loadConfig())

	log.Infof("Using max file size: %dMB\n", config.MaxFileSizeMB)

	log.Info("Connecting to database...")
	log.MustPanic(connectToDatabase())

	handleTasks()

	app := fiber.New(fiber.Config{
		BodyLimit: int(config.MaxFileSize()),
	})

	app.Use(recover.New())
	app.Use(adapter.FiberMiddleware(log))
	app.Use(authenticate)

	app.Post("/upload", uploadHandler)
	app.Get("/echos", listEchosHandler)
	app.Delete("/echos/:hash", deleteEchoHandler)

	log.Infof("Starting server at %s...\n", config.Addr())
	log.MustPanic(app.Listen(config.Addr()))
}
