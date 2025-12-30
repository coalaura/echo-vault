package main

import (
	"encoding/json"
	"net/http"
)

func abort(w http.ResponseWriter, code int, err string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)

	json.NewEncoder(w).Encode(map[string]string{
		"error": err,
	})
}

func okay(w http.ResponseWriter, ct ...string) {
	if len(ct) == 1 {
		w.Header().Add("Content-Type", ct[0])
	}

	w.WriteHeader(http.StatusOK)
}
