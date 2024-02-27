package custommiddleware

import (
	"net/http"
)

func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Auth middleware hit!"))
		next.ServeHTTP(w, r)
	})
}
