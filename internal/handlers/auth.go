package handlers

import (
	"errors"
	"log"
	"net/http"
)

// TODO
// read session cookie and authenticate if valid

func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Cookie("lifeDashboard")
		if err != nil {
			switch {
			case errors.Is(err, http.ErrNoCookie):
				http.Error(w, "cookie not found", http.StatusBadRequest)
			default:
				log.Println(err)
				http.Error(w, "server error", http.StatusInternalServerError)
			}
			return
		}

		// token, err := utils.VerifyToken(cookie.Value)
		// if err != nil {
		// 	http.Error(w, "not authorised", http.StatusUnauthorized)
		// 	return
		// }

		w.Write([]byte("Auth middleware hit!\n"))
		next.ServeHTTP(w, r)
	})
}
