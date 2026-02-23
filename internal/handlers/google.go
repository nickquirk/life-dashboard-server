package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/nickquirk/life-dashboard-server/internal/config"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/utils"
	"golang.org/x/oauth2"
)

// generateStateOauthCookie creates a random OAuth state and sets it as a cookie via CookieConfig.
func (h *Handler) generateStateOauthCookie(w http.ResponseWriter) string {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	http.SetCookie(w, h.Cookies.NewOAuthStateCookie(state))
	return state
}

func (h *Handler) googleLogin(w http.ResponseWriter, r *http.Request) {
	oauthstate := h.generateStateOauthCookie(w)

	url := config.GoogleConfiguration.GoogleLoginConfig.AuthCodeURL(
		oauthstate,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *Handler) googleCallback(w http.ResponseWriter, r *http.Request) {
	oauthStateCookie, err := r.Cookie("oauthstate")
	if err != nil {
		http.Error(w, "OAuth state cookie missing", http.StatusBadRequest)
		slog.Error("oauth state cookie missing", "error", err)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "State param not set or empty", http.StatusUnauthorized)
		slog.Error("state param not set or empty", "error", err)
		return
	}

	if state != oauthStateCookie.Value {
		http.Error(w, "Google Auth states do not match", http.StatusUnauthorized)
		slog.Warn("oauth state mismatch, potential CSRF")
		return
	}

	// Clear state cookie now that its been used
	http.SetCookie(w, h.Cookies.ExpireOAuthStateCookie())

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		slog.Error("missing authorization code", "error", err)
		return
	}
	googleConfig := config.GetGoogleConfig()
	ctx := context.Background()

	token, err := googleConfig.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "Code-Token exchange failed", http.StatusInternalServerError)
		slog.Error("code-token exchange failed", "error", err)
		return
	}

	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		http.Error(w, "User data fetch failed", http.StatusInternalServerError)
		slog.Error("user data fetch failed", "error", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to fetch user info from Google", http.StatusInternalServerError)
		slog.Error("failed to fetch user info from Google", "error", err)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to parse Google response body", http.StatusInternalServerError)
		slog.Error("failed to parse google response body", "error", err)
		return
	}

	// Add oauth2 tokens and expiry to userData
	userData := domain.CreateUserRequest{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenExpiry:  token.Expiry,
	}

	err = json.Unmarshal(body, &userData)
	if err != nil {
		http.Error(w, "Failed to unmarshal user data", http.StatusInternalServerError)
		slog.Error("failed to unmarshal user data", "error", err)
		return
	}

	// Save user to DB
	user, err := h.CreateUser(userData)
	if err != nil {
		slog.Error("failed to save user to database", "error", err)
		http.Error(w, "Failed to save user", http.StatusInternalServerError)
		return
	}

	// save id and email in JWT
	tok, err := utils.GenerateToken(user.Id, userData.Email)
	if err != nil {
		slog.Error("failed to generate JWT", "error", err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	// Generate app-level refresh token
	rawRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		slog.Error("failed to generate refresh token", "error", err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}
	hashedRefreshToken := utils.HashRefreshToken(rawRefreshToken)
	if err := h.Service.UpdateAppRefreshToken(user.Id, hashedRefreshToken); err != nil {
		slog.Error("failed to store refresh token", "error", err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, h.Cookies.NewSessionCookie(tok))
	http.SetCookie(w, h.Cookies.NewRefreshCookie(rawRefreshToken))

	clientURL := os.Getenv("CLIENT_URL")
	http.Redirect(w, r, clientURL+"/?view=triage", http.StatusTemporaryRedirect)
}
