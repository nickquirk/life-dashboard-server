package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

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
		h.respondWithError(w, "OAuth state cookie missing", err, http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		h.respondWithError(w, "State param not set or empty", fmt.Errorf("missing state query parameter"), http.StatusUnauthorized)
		return
	}

	if state != oauthStateCookie.Value {
		h.respondWithError(w, "Google Auth states do not match", fmt.Errorf("oauth state mismatch, potential CSRF"), http.StatusUnauthorized)
		return
	}

	// Clear state cookie now that its been used
	http.SetCookie(w, h.Cookies.ExpireOAuthStateCookie())

	code := r.URL.Query().Get("code")
	if code == "" {
		h.respondWithError(w, "Missing authorization code", fmt.Errorf("missing authorization code query parameter"), http.StatusBadRequest)
		return
	}
	googleConfig := config.GetGoogleConfig()
	ctx := context.Background()

	token, err := googleConfig.Exchange(ctx, code)
	if err != nil {
		h.respondWithError(w, "Code-Token exchange failed", err, http.StatusInternalServerError)
		return
	}

	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		h.respondWithError(w, "User data fetch failed", err, http.StatusInternalServerError)
		return
	}

	// to avoid leaking connections
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.respondWithError(w, "Failed to fetch user info from Google", fmt.Errorf("google userinfo returned status %d", resp.StatusCode), http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.respondWithError(w, "Failed to parse Google response body", err, http.StatusInternalServerError)
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
		h.respondWithError(w, "Failed to unmarshal user data", err, http.StatusInternalServerError)
		return
	}

	// Save user to DB
	user, err := h.Service.CreateUser(userData)
	if err != nil {
		h.respondWithError(w, "Failed to save user", err, http.StatusInternalServerError)
		return
	}

	// save id and email in JWT
	tok, err := utils.GenerateToken(user.ID, userData.Email)
	if err != nil {
		h.respondWithError(w, "Authentication failed", err, http.StatusInternalServerError)
		return
	}

	// Generate app-level refresh token
	rawRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		h.respondWithError(w, "Authentication failed", err, http.StatusInternalServerError)
		return
	}
	hashedRefreshToken := utils.HashRefreshToken(rawRefreshToken)
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // Match cookie expiry of 30 days

	if err := h.Service.CreateSession(user.ID, hashedRefreshToken, expiresAt, r.UserAgent()); err != nil {
		h.respondWithError(w, "Authentication failed", err, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, h.Cookies.NewSessionCookie(tok))
	http.SetCookie(w, h.Cookies.NewRefreshCookie(rawRefreshToken))

	clientURL := os.Getenv("CLIENT_URL")
	http.Redirect(w, r, clientURL, http.StatusTemporaryRedirect)
}
