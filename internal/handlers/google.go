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

func (h *Handler) googleLogin(w http.ResponseWriter, r *http.Request) {
	oauthstate := generateStateOauthCookie(w)

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

	params := r.URL.Query()
	state := params["state"][0]

	if state != oauthStateCookie.Value {
		http.Error(w, "Google Auth states do not match", http.StatusUnauthorized)
		slog.Warn("oauth state mismatch, potential CSRF")
		return
	}

	// Clear state cookie now that its been used
	clearCookie := http.Cookie{
		Name:     "oauthstate",
		Value:    "",
		MaxAge:   -1, // Deletes the cookie
		HttpOnly: true,
		Secure:   os.Getenv("ENV") == "prod",
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	}
	http.SetCookie(w, &clearCookie)

	code := params["code"][0]
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

	isProd := os.Getenv("ENV") == "prod"

	http.SetCookie(w, &http.Cookie{
		Name:     "life-dashboard",
		Value:    tok,
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "life-dashboard-refresh",
		Value:    rawRefreshToken,
		HttpOnly: true,
		Secure:   isProd,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
	})

	clientURL := os.Getenv("CLIENT_URL")
	http.Redirect(w, r, clientURL+"/?view=triage", http.StatusTemporaryRedirect)
}

// Helper to generate a random state and set it as a cookie
func generateStateOauthCookie(w http.ResponseWriter) string {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	cookie := http.Cookie{
		Name:     "oauthstate",
		Value:    state,
		MaxAge:   int(10 * 60),               // 10 minutes is plenty of time to log in
		HttpOnly: true,                       // Protects against XSS
		Secure:   os.Getenv("ENV") == "prod", // Secure over HTTPS in prod
		SameSite: http.SameSiteLaxMode,       // Required for the cross-site redirect callback
		Path:     "/",
	}
	http.SetCookie(w, &cookie)

	return state
}
