package handlers

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"

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

	// Best-effort: invalidate refresh token in DB if we can identify the user
	if jwtCookie, err := r.Cookie("life-dashboard"); err == nil {
		if userID, err := utils.GetUserIdFromExpiredToken(jwtCookie.Value); err == nil {
			if err := h.Service.UpdateAppRefreshToken(userID, ""); err != nil {
				slog.Warn("failed to invalidate refresh token on logout", "error", err, "userID", userID)
			}
			slog.Info("user logged out", "userID", userID)
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

	//  Get the refresh token from cookie
	refreshCookie, err := r.Cookie("life-dashboard-refresh")
	if err != nil {
		h.respondWithError(w, "Unauthorized", err, http.StatusUnauthorized, "userID", userID)
		return
	}

	// Validate refresh token against stored hash
	providedHash := utils.HashRefreshToken(refreshCookie.Value)
	storedHash, err := h.Service.GetAppRefreshToken(userID)
	if err != nil || storedHash == "" {
		if err == nil {
			err = fmt.Errorf("no stored refresh token")
		}
		h.respondWithError(w, "Unauthorized", err, http.StatusUnauthorized, "userID", userID)
		return
	}

	if subtle.ConstantTimeCompare([]byte(providedHash), []byte(storedHash)) != 1 {
		h.respondWithError(w, "Unauthorized", fmt.Errorf("refresh token hash mismatch"), http.StatusUnauthorized, "userID", userID)
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

	// Rotate refresh token
	newRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		h.respondWithError(w, "Token refresh failed", err, http.StatusInternalServerError)
		return
	}

	newHash := utils.HashRefreshToken(newRefreshToken)
	if err := h.Service.UpdateAppRefreshToken(userID, newHash); err != nil {
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
