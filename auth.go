package main

import (
	"net/http"
	"strings"
)

func authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAuthenticated(r) {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAuthenticated(r *http.Request) bool {
	if config.Server.UploadToken == "" {
		return true
	}

	token := r.Header.Get("Authorization")

	if !strings.HasPrefix(token, "Bearer ") {
		return false
	}

	token = strings.TrimPrefix(token, "Bearer ")

	return token == config.Server.UploadToken
}
