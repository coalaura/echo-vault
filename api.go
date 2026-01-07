package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mozillazg/go-unidecode"
)

const PageSize = 100

func infoHandler(w http.ResponseWriter, r *http.Request) {
	okay(w, "application/json")

	json.NewEncoder(w).Encode(map[string]any{
		"version": Version,
		"queries": config.AI.OpenRouterToken != "",
	})
}

func verifyHandler(w http.ResponseWriter, r *http.Request) {
	if isAuthenticated(r) {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
	}
}

func viewEchoHandler(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	if !validateHash(hash) {
		abort(w, http.StatusBadRequest, "invalid hash format")

		log.Warnln("view: invalid hash")

		return
	}

	ext := chi.URLParam(r, "ext")
	if !config.IsValidImageFormat(ext) && !config.IsValidVideoFormat(ext, false) {
		abort(w, http.StatusBadRequest, "invalid extension")

		log.Warnln("view: invalid extension")

		return
	}

	storage, err := storageAbs()
	if err != nil {
		abort(w, http.StatusInternalServerError, "storage configuration error")

		log.Warnln("view: failed to resolve storage")
		log.Warnln(err)

		return
	}

	path := filepath.Join(storage, hash+"."+ext)

	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			abort(w, http.StatusNotFound, "echo not found")

			return
		}

		abort(w, http.StatusInternalServerError, "failed to read echo file")

		log.Warnln("view: failed to open file")
		log.Warnln(err)

		return
	}

	defer file.Close()

	w.Header().Set("Cache-Control", "public, max-age=604800, must-revalidate")

	okay(w)

	io.Copy(w, file)
}

func listEchosHandler(w http.ResponseWriter, r *http.Request) {
	page := parsePage(r)
	if page == -1 {
		abort(w, http.StatusBadRequest, "invalid page number")

		log.Warnln("list: invalid page number")

		return
	}

	echos, err := database.FindAll((page-1)*PageSize, PageSize)
	if err != nil {
		abort(w, http.StatusInternalServerError, "database error")

		log.Warnln("list: failed to read echos")
		log.Warnln(err)

		return
	}

	okay(w, "application/json")

	json.NewEncoder(w).Encode(map[string]any{
		"echos": echos,
		"size":  usage.Load(),
		"count": count.Load(),
	})
}

func deleteEchoHandler(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	if !validateHash(hash) {
		abort(w, http.StatusBadRequest, "invalid hash format")

		log.Warnln("delete: invalid hash")

		return
	}

	echo, err := database.Find(hash)
	if err != nil {
		abort(w, http.StatusInternalServerError, "database error")

		log.Warnln("delete: failed to find echo")
		log.Warnln(err)

		return
	}

	if echo == nil {
		abort(w, http.StatusNotFound, "echo not found")

		log.Warnf("delete: echo %q not found\n", hash)

		return
	}

	err = echo.Unlink()
	if err != nil {
		abort(w, http.StatusInternalServerError, "filesystem error")

		log.Warnln("delete: failed to unlink echo")
		log.Warnln(err)

		return
	}

	err = database.Delete(hash)
	if err != nil {
		abort(w, http.StatusInternalServerError, "database delete error")

		log.Warnln("delete: failed to delete echo")
		log.Warnln(err)

		return
	}

	if vector != nil {
		vector.Delete(hash)
	}

	count.Add(^uint64(0))

	okay(w)
}

func queryEchosHandler(w http.ResponseWriter, r *http.Request) {
	if vector == nil {
		abort(w, http.StatusServiceUnavailable, "querying is disabled")

		return
	}

	page := parsePage(r)
	if page == -1 {
		abort(w, http.StatusBadRequest, "invalid page number")

		log.Warnln("query: invalid page number")

		return
	}

	query := r.URL.Query().Get("q")

	query = strings.TrimSpace(query)
	query = unidecode.Unidecode(query)

	if len(query) == 0 {
		abort(w, http.StatusBadRequest, "missing query")

		log.Warnln("query: missing query")

		return
	}

	ranked, err := vector.Query(r.Context(), query, page*PageSize)
	if err != nil {
		abort(w, http.StatusInternalServerError, "failed search")

		log.Warnln("query: failed search")
		log.Warnln(err)

		return
	}

	var echos []Echo

	start := (page - 1) * PageSize
	end := min(start+PageSize, len(ranked))

	if start < end {
		ranked = ranked[start:end]

		hashes := make([]string, len(ranked))
		scoreMap := make(map[string]float32)

		for i, res := range ranked {
			hashes[i] = res.Hash

			scoreMap[res.Hash] = res.Similarity
		}

		results, err := database.FindByHashes(hashes)
		if err != nil {
			abort(w, http.StatusInternalServerError, "database error")

			log.Warnln("query: failed to read echos")
			log.Warnln(err)

			return
		}

		for i, echo := range results {
			if score, ok := scoreMap[echo.Hash]; ok {
				results[i].Tag.Similarity = score
			}
		}

		echos = results
	}

	okay(w, "application/json")

	json.NewEncoder(w).Encode(map[string]any{
		"echos": echos,
		"size":  usage.Load(),
		"count": count.Load(),
	})
}

func parsePage(r *http.Request) int {
	var page int

	if raw := chi.URLParam(r, "page"); raw != "" {
		num, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return -1
		}

		page = int(num)
	}

	return max(1, page)
}
