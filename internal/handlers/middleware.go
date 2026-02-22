package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/utils"
)

func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the cookie
		cookie, err := r.Cookie("life-dashboard") // Ensure name matches SetCookie in google.go
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate token and get User ID
		userID, err := utils.GetUserIdFromToken(cookie.Value)
		if err != nil {
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
			return
		}

		// Store UserID in context so the handler can use it
		ctx := context.WithValue(r.Context(), UserIDKey, userID)

		// Call the next handler with the new context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// To give extra time to slow requests, eg syncUsers
func WithExtendedTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rc := http.NewResponseController(w)
			rc.SetWriteDeadline(time.Now().Add(timeout))
			next.ServeHTTP(w, r)
		})
	}
}
