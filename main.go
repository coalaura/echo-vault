package main

import (
	"embed"
	_ "embed"
	"io/fs"
	"net/http"
	"os"

	"github.com/coalaura/plain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var Version = "dev"

var (
	config   *EchoConfig
	database *EchoDatabase

	//go:embed public/*
	publicFs embed.FS

	log = plain.New(plain.WithDate(plain.RFC3339Local))
)

func main() {
	log.Printf("Echo-Vault %s\n", Version)

	err := os.MkdirAll("./storage", 0755)
	log.MustFail(err)

	public, err := fs.Sub(publicFs, "public")
	log.MustFail(err)

	log.Println("Loading config...")

	config, err = LoadConfig()
	log.MustFail(err)

	log.Println("Connecting to database...")

	database, err = ConnectToDatabase()
	log.MustFail(err)

	handleTasks()

	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(log.Middleware())

	fs := http.FileServer(http.FS(public))
	r.Handle("/*", fs)

	r.Get("/info", infoHandler)
	r.Get("/verify", verifyHandler)

	r.Group(func(gr chi.Router) {
		gr.Use(authenticate)

		gr.Post("/upload", uploadHandler)
		gr.Get("/echos/{page}", listEchosHandler)
		gr.Delete("/echos/{hash}", deleteEchoHandler)
	})

	r.Get("/i/{hash}.{ext}", viewEchoHandler)

	addr := config.Addr()

	log.Printf("Listening at http://localhost%s/\n", addr)
	http.ListenAndServe(addr, r)
}
