package main

import (
	"os"

	"github.com/coalaura/plain"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

var (
	config   EchoConfig
	database *EchoDatabase

	log = plain.New(plain.WithDate(plain.RFC3339Local))
)

func main() {
	os.MkdirAll("./storage", 0755)

	log.Println("Loading env...")
	log.MustFail(loadConfig())

	log.Printf("Using max file size: %dMB\n", config.MaxFileSizeMB)

	log.Println("Connecting to database...")
	log.MustFail(connectToDatabase())

	handleTasks()

	app := fiber.New(fiber.Config{
		BodyLimit: int(config.MaxFileSize()),
	})

	app.Use(recover.New())
	app.Use(log.Middleware())
	app.Use(authenticate)

	app.Post("/upload", uploadHandler)
	app.Get("/echos", listEchosHandler)
	app.Delete("/echos/:hash", deleteEchoHandler)

	log.Printf("Starting server at %s...\n", config.Addr())
	log.MustFail(app.Listen(config.Addr()))
}
