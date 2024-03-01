package handlers

import (
	"net/http"
)

// TODO
// read session cookie and authenticate if valid

func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Auth middleware hit!\n"))
		next.ServeHTTP(w, r)
	})
}
