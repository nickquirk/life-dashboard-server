package handlers

import (
	"context"
	"net/http"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/utils"
	"golang.org/x/oauth2"
)

// TODO
// read session cookie and authenticate if valid

type contextKey string

const UserIDKey contextKey = "userID"

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

// Retrieve a token, saves the token, then returns the generated client.
func (h *Handler) GetClient(config *oauth2.Config) (*http.Client, error) {
	userToLogIn := domain.GetUserRequest{}

	user, err := h.GetUser(userToLogIn)
	if err != nil {
		return config.Client(context.Background(), &oauth2.Token{}), err
	}

	tok := &oauth2.Token{
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
		Expiry:       user.TokenExpiry,
	}

	return config.Client(context.Background(), tok), nil
}
