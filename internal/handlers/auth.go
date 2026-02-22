package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/utils"
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
	isProd := os.Getenv("ENV") == "prod"

	// Clear JWT cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "life-dashboard",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode,
	})

	// Clear refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "life-dashboard-refresh",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode,
	})

	// Best-effort: invalidate refresh token in DB if we can identify the user
	if jwtCookie, err := r.Cookie("life-dashboard"); err == nil {
		if userID, err := utils.GetUserIdFromExpiredToken(jwtCookie.Value); err == nil {
			_ = h.Service.UpdateAppRefreshToken(userID, "")
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Logged out"}`))
}

func (h *Handler) refreshToken(w http.ResponseWriter, r *http.Request) {
	// 1. Get user ID from the expired JWT
	jwtCookie, err := r.Cookie("life-dashboard")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := utils.GetUserIdFromExpiredToken(jwtCookie.Value)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// 2. Get the refresh token from cookie
	refreshCookie, err := r.Cookie("life-dashboard-refresh")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 3. Validate refresh token against stored hash
	providedHash := utils.HashRefreshToken(refreshCookie.Value)
	storedHash, err := h.Service.GetAppRefreshToken(userID)
	if err != nil || storedHash == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if subtle.ConstantTimeCompare([]byte(providedHash), []byte(storedHash)) != 1 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 4. Get user info for new JWT claims
	user, err := h.Service.GetUser(domain.GetUserRequest{Id: userID})
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 5. Generate new JWT
	newJWT, err := utils.GenerateToken(userID, user.Email)
	if err != nil {
		log.Printf("Failed to generate JWT during refresh: %v", err)
		http.Error(w, "Token refresh failed", http.StatusInternalServerError)
		return
	}

	// 6. Rotate refresh token
	newRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		log.Printf("Failed to generate refresh token during refresh: %v", err)
		http.Error(w, "Token refresh failed", http.StatusInternalServerError)
		return
	}

	newHash := utils.HashRefreshToken(newRefreshToken)
	if err := h.Service.UpdateAppRefreshToken(userID, newHash); err != nil {
		log.Printf("Failed to store rotated refresh token: %v", err)
		http.Error(w, "Token refresh failed", http.StatusInternalServerError)
		return
	}

	// 7. Set both cookies
	isProd := os.Getenv("ENV") == "prod"

	http.SetCookie(w, &http.Cookie{
		Name:     "life-dashboard",
		Value:    newJWT,
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "life-dashboard-refresh",
		Value:    newRefreshToken,
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
	})

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
