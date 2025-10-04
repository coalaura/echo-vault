package main

import (
	"net/http"
	"os"

	"github.com/coalaura/plain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var Version = "dev"

var (
	config   EchoConfig
	database *EchoDatabase

	log = plain.New(plain.WithDate(plain.RFC3339Local))
)

func main() {
	log.Printf("Echo-Vault %s\n", Version)

	os.MkdirAll("./storage", 0755)

	log.Println("Loading env...")
	log.MustFail(loadConfig())

	log.Printf("Using max file size: %dMB\n", config.Server.MaxFileSize)

	log.Println("Connecting to database...")
	log.MustFail(connectToDatabase())

	handleTasks()

	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(log.Middleware())

	r.Group(func(gr chi.Router) {
		gr.Use(authenticate)

		gr.Post("/upload", uploadHandler)
		gr.Get("/echos/{page}", listEchosHandler)
		gr.Delete("/echos/{hash}", deleteEchoHandler)
	})

	r.Get("/favicon.ico", viewEchoHandler)
	r.Get("/{hash}.{ext}", viewEchoHandler)

	addr := config.Addr()

	log.Printf("Listening at %s\n", addr)
	http.ListenAndServe(addr, r)
}
