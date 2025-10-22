package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func viewEchoHandler(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	if !validateHash(hash) {
		abort(w, http.StatusBadRequest)

		log.Warnln("view: invalid hash")

		return
	}

	ext := chi.URLParam(r, "ext")
	if !config.IsValidImageFormat(ext) && !config.IsValidVideoFormat(ext, false) {
		abort(w, http.StatusBadRequest)

		log.Warnln("view: invalid extension")

		return
	}

	storage, err := storageAbs()
	if err != nil {
		abort(w, http.StatusInternalServerError)

		log.Warnln("view: failed to resolve storage")
		log.Warnln(err)

		return
	}

	path := filepath.Join(storage, hash+"."+ext)

	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			abort(w, http.StatusNotFound)

			return
		}

		log.Warnln("view: failed to open file")
		log.Warnln(err)

		return
	}

	defer file.Close()

	okay(w)

	io.Copy(w, file)
}

func listEchosHandler(w http.ResponseWriter, r *http.Request) {
	var page int

	if raw := chi.URLParam(r, "page"); raw != "" {
		num, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			abort(w, http.StatusBadRequest)

			log.Warnln("list: invalid page number")
			log.Warnln(err)

			return
		}

		page = int(num)
	}

	page = max(1, page)

	echos, err := database.FindAll((page-1)*15, 15)
	if err != nil {
		abort(w, http.StatusInternalServerError)

		log.Warnln("list: failed to read echos")
		log.Warnln(err)

		return
	}

	okay(w, "application/json")

	json.NewEncoder(w).Encode(echos)
}

func deleteEchoHandler(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	if !validateHash(hash) {
		abort(w, http.StatusBadRequest)

		log.Warnln("delete: invalid hash")

		return
	}

	echo, err := database.Find(hash)
	if err != nil {
		abort(w, http.StatusInternalServerError)

		log.Warnln("delete: failed to find echo")
		log.Warnln(err)

		return
	}

	if echo == nil {
		abort(w, http.StatusNotFound)

		log.Warnf("delete: echo %q not found\n", hash)

		return
	}

	err = echo.Unlink()
	if err != nil {
		abort(w, http.StatusInternalServerError)

		log.Warnln("delete: failed to unlink echo")
		log.Warnln(err)

		return
	}

	err = database.Delete(hash)
	if err != nil {
		abort(w, http.StatusInternalServerError)

		log.Warnln("delete: failed to delete echo")
		log.Warnln(err)

		return
	}

	okay(w)
}
