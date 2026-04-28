package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time" // Added for session expiration

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/utils"
	"golang.org/x/oauth2"
)

type contextKey string

const UserIDKey contextKey = "userID"

// Retrieve a token, saves the token, then returns the generated client.
func (h *Handler) GetClient(config *oauth2.Config) (*http.Client, error) {
	userToLogIn := domain.GetUserRequest{}

	user, err := h.Service.GetUser(userToLogIn)
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
	http.SetCookie(w, h.Cookies.ExpireSessionCookie())
	http.SetCookie(w, h.Cookies.ExpireRefreshCookie())

	// Best-effort: invalidate the specific session in the DB using the refresh token
	if refreshCookie, err := r.Cookie("life-dashboard-refresh"); err == nil {
		if jwtCookie, err := r.Cookie("life-dashboard"); err == nil {
			if userID, err := utils.GetUserIdFromExpiredToken(jwtCookie.Value); err == nil {
				hashedToken := utils.HashRefreshToken(refreshCookie.Value)

				// Delete this specific session, leaving other devices untouched
				if err := h.Service.DeleteSession(userID, hashedToken); err != nil {
					slog.Warn("failed to delete session on logout", "error", err, "userID", userID)
				}
				slog.Info("user logged out", "userID", userID)
			}
		}
	}

	h.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Logged out"})
}

func (h *Handler) refreshToken(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the expired JWT
	jwtCookie, err := r.Cookie("life-dashboard")
	if err != nil {
		h.respondWithError(w, "Unauthorized", err, http.StatusUnauthorized)
		return
	}

	userID, err := utils.GetUserIdFromExpiredToken(jwtCookie.Value)
	if err != nil {
		h.respondWithError(w, "Invalid token", err, http.StatusUnauthorized)
		return
	}

	// Get the refresh token from cookie
	refreshCookie, err := r.Cookie("life-dashboard-refresh")
	if err != nil {
		h.respondWithError(w, "Unauthorized", err, http.StatusUnauthorized, "userID", userID)
		return
	}

	// Hash the incoming token
	providedHash := utils.HashRefreshToken(refreshCookie.Value)

	// Validate the session exists and hasn't expired
	isValid, err := h.Service.ValidateSession(userID, providedHash)
	if err != nil || !isValid {
		if err == nil {
			err = fmt.Errorf("invalid or expired refresh token")
		}
		h.respondWithError(w, "Unauthorized", err, http.StatusUnauthorized, "userID", userID)
		return
	}

	// Get user email for new JWT claims
	email, err := h.Service.GetUserEmail(userID)
	if err != nil {
		h.respondWithError(w, "Unauthorized", err, http.StatusUnauthorized, "userID", userID)
		return
	}

	// Generate new JWT
	newJWT, err := utils.GenerateToken(userID, email)
	if err != nil {
		h.respondWithError(w, "Token refresh failed", err, http.StatusInternalServerError)
		return
	}

	// Generate a new refresh token (rotation)
	newRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		h.respondWithError(w, "Token refresh failed", err, http.StatusInternalServerError)
		return
	}

	// Rotate the session: Delete the old one and create a new one
	_ = h.Service.DeleteSession(userID, providedHash)

	newHash := utils.HashRefreshToken(newRefreshToken)
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // Match your cookie expiry of 30 days

	if err := h.Service.CreateSession(userID, newHash, expiresAt); err != nil {
		h.respondWithError(w, "Token refresh failed", err, http.StatusInternalServerError)
		return
	}

	// Set both cookies
	http.SetCookie(w, h.Cookies.NewSessionCookie(newJWT))
	http.SetCookie(w, h.Cookies.NewRefreshCookie(newRefreshToken))

	h.respondWithJSON(w, http.StatusOK, map[string]string{"message": "Token refreshed"})
}

// Extract the user ID from the request context.
// It returns (0, false) if the ID is missing or invalid.
func (h *Handler) GetUserID(r *http.Request) (uint, bool) {
	userID, ok := r.Context().Value(UserIDKey).(uint)
	return userID, ok
}
