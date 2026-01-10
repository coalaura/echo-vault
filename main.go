package main

import (
	"context"
	"embed"
	_ "embed"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/coalaura/plain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var Version = "dev"

var (
	config   *EchoConfig
	database *EchoDatabase
	vector   *VectorStore
	usage    atomic.Uint64
	count    atomic.Uint64

	//go:embed public/*
	publicFs embed.FS

	log = plain.New(plain.WithDate(plain.RFC3339Local))
)

func main() {
	log.Printf("Echo-Vault %s\n", Version)

	err := EnsureStorage()
	log.MustFail(err)

	public, err := getPublicFS()
	log.MustFail(err)

	log.Println("Loading config...")

	config, err = LoadConfig()
	log.MustFail(err)

	log.Println("Loading vector store...")

	vector, err = LoadVectorStore()
	log.MustFail(err)

	log.Println("Connecting to database...")

	database, err = ConnectToDatabase()
	log.MustFail(err)

	defer database.Close()

	size, total, err := database.Verify()
	log.MustFail(err)

	err = StartBackupLoop()
	log.MustFail(err)

	usage.Add(size)
	count.Add(total)

	go RunBackfill(total)

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
		gr.Get("/query/{page}", queryEchosHandler)
		gr.Put("/echos/{hash}", updateEchoHandler)
		gr.Delete("/echos/{hash}", deleteEchoHandler)
	})

	r.Get("/i/{hash}.{ext}", viewEchoHandler)

	addr := config.Addr()

	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		log.Printf("Listening at http://localhost%s/\n", addr)

		err = server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Warnln(err)
		}
	}()

	log.WaitForInterrupt(true)

	log.Warnln("Shutting down...")

	shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	server.Shutdown(shutdown)
}

func getPublicFS() (fs.FS, error) {
	if Version == "dev" {
		return os.DirFS("public"), nil
	}

	return fs.Sub(publicFs, "public")
}
