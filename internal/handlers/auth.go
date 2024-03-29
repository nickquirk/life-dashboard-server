package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/nickquirk/life-dashboard-server/internal/utils"
)

// TODO
// read session cookie and authenticate if valid

func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("lifeDashboard")
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

		authToken, err := utils.VerifyToken(cookie.Value)
		if err != nil {
			http.Error(w, "not authorised", http.StatusUnauthorized)
			fmt.Printf("error: %v\n", err)
			return
		}

		fmt.Printf("auth tok: %s", authToken)

		//utils.SaveToken("token.json", authToken)

		w.Write([]byte("Auth middleware hit!\n"))
		next.ServeHTTP(w, r)
	})
}
