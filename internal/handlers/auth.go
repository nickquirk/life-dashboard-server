package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
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

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Logged out"}`))
}

func (h *Handler) refreshToken(w http.ResponseWriter, r *http.Request) {
	// Get user ID from the expired JWT
	jwtCookie, err := r.Cookie("life-dashboard")
	if err != nil {
		slog.Warn("refresh token: session cookie missing", "error", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := utils.GetUserIdFromExpiredToken(jwtCookie.Value)
	if err != nil {
		slog.Warn("refresh token: invalid expired JWT", "error", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	//  Get the refresh token from cookie
	refreshCookie, err := r.Cookie("life-dashboard-refresh")
	if err != nil {
		slog.Warn("refresh token: refresh cookie missing", "userID", userID)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Validate refresh token against stored hash
	providedHash := utils.HashRefreshToken(refreshCookie.Value)
	storedHash, err := h.Service.GetAppRefreshToken(userID)
	if err != nil || storedHash == "" {
		slog.Warn("refresh token: stored refresh token not found", "error", err, "userID", userID)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if subtle.ConstantTimeCompare([]byte(providedHash), []byte(storedHash)) != 1 {
		slog.Warn("refresh token: hash mismatch", "userID", userID)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user email for new JWT claims
	email, err := h.Service.GetUserEmail(userID)
	if err != nil {
		slog.Warn("refresh token: failed to get user email", "error", err, "userID", userID)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Generate new JWT
	newJWT, err := utils.GenerateToken(userID, email)
	if err != nil {
		slog.Error("failed to generate JWT during refresh", "error", err)
		http.Error(w, "Token refresh failed", http.StatusInternalServerError)
		return
	}

	// Rotate refresh token
	newRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		slog.Error("failed to generate refresh token during refresh", "error", err)
		http.Error(w, "Token refresh failed", http.StatusInternalServerError)
		return
	}

	newHash := utils.HashRefreshToken(newRefreshToken)
	if err := h.Service.UpdateAppRefreshToken(userID, newHash); err != nil {
		slog.Error("failed to store rotated refresh token", "error", err)
		http.Error(w, "Token refresh failed", http.StatusInternalServerError)
		return
	}

	// Set both cookies
	http.SetCookie(w, h.Cookies.NewSessionCookie(newJWT))
	http.SetCookie(w, h.Cookies.NewRefreshCookie(newRefreshToken))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Token refreshed"})
}

// Extract the user ID from the request context.
// It returns (0, false) if the ID is missing or invalid.
func (h *Handler) GetUserID(r *http.Request) (uint, bool) {
	userID, ok := r.Context().Value(UserIDKey).(uint)
	return userID, ok
}
