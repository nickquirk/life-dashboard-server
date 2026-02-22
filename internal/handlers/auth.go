package handlers

import (
	"context"
	"net/http"
	"os"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"golang.org/x/oauth2"
)

type contextKey string

const UserIDKey contextKey = "userID"

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

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	// Overwrite the cookie with one that expires immediately
	cookie := http.Cookie{
		Name:     "life-dashboard",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Tell browser to delete it
		HttpOnly: true,
		Secure:   os.Getenv("ENV") == "prod",
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, &cookie)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Logged out"}`))
}

// Extract the user ID from the request context.
// It returns (0, false) if the ID is missing or invalid.
func (h *Handler) GetUserID(r *http.Request) (uint, bool) {
	userID, ok := r.Context().Value(UserIDKey).(uint)
	return userID, ok
}
