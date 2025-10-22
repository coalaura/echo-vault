package main

import (
	"net/http"
)

func abort(w http.ResponseWriter, code int) {
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(code)

	w.Write([]byte(http.StatusText(code)))
}

func okay(w http.ResponseWriter, ct ...string) {
	if len(ct) == 1 {
		w.Header().Add("Content-Type", ct[0])
	}

	w.WriteHeader(http.StatusOK)
}
