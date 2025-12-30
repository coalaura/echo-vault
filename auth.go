package main

import (
	"net/http"
	"strings"
)

func authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAuthenticated(r) {
			abort(w, http.StatusUnauthorized, "unauthorized")

			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAuthenticated(r *http.Request) bool {
	token := r.Header.Get("Authorization")

	if !strings.HasPrefix(token, "Bearer ") {
		return false
	}

	token = strings.TrimPrefix(token, "Bearer ")

	return token == config.Server.UploadToken
}
