package handlers

import (
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

const (
	authErrLoginExpired      = "login_expired"
	authErrGoogleFailed      = "google_failed"
	authErrServerUnavailable = "server_unavailable"
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
		h.redirectWithAuthError(w, r, authErrLoginExpired, "OAuth state cookie missing", err)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		h.redirectWithAuthError(w, r, authErrLoginExpired, "State param not set or empty", fmt.Errorf("missing state query parameter"))
		return
	}

	if state != oauthStateCookie.Value {
		h.redirectWithAuthError(w, r, authErrLoginExpired, "Google Auth states do not match", fmt.Errorf("oauth state mismatch, potential CSRF"))
		return
	}

	// Clear state cookie now that its been used
	http.SetCookie(w, h.Cookies.ExpireOAuthStateCookie())

	// User clicked Cancel on Google's consent screen: not an error
	if r.URL.Query().Get("error") == "access_denied" {
		http.Redirect(w, r, os.Getenv("CLIENT_URL"), http.StatusSeeOther)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		h.redirectWithAuthError(w, r, authErrGoogleFailed, "Missing authorization code",
			fmt.Errorf("missing authorization code query parameter (google error=%q)", r.URL.Query().Get("error")))
		return
	}

	googleConfig := config.GetGoogleConfig()
	ctx := r.Context()

	token, err := googleConfig.Exchange(ctx, code)
	if err != nil {
		h.redirectWithAuthError(w, r, authErrGoogleFailed, "Code-Token exchange failed", err)
		return
	}

	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		h.redirectWithAuthError(w, r, authErrGoogleFailed, "User data fetch failed", err)
		return
	}

	// to avoid leaking connections
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.redirectWithAuthError(w, r, authErrGoogleFailed, "Failed to fetch user info from Google",
			fmt.Errorf("google userinfo returned status %d", resp.StatusCode))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.redirectWithAuthError(w, r, authErrGoogleFailed, "Failed to read Google response body", err)
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
		h.redirectWithAuthError(w, r, authErrGoogleFailed, "Failed to unmarshal user data", err)
		return
	}

	// Save user to DB
	user, err := h.Service.CreateUser(userData)
	if err != nil {
		h.redirectWithAuthError(w, r, authErrServerUnavailable, "Failed to save user", err, "email", userData.Email)
		return
	}

	// save id and email in JWT
	tok, err := utils.GenerateToken(user.ID, userData.Email)
	if err != nil {
		h.redirectWithAuthError(w, r, authErrServerUnavailable, "Failed to generate session token", err, "userID", user.ID)
		return
	}

	// Generate app-level refresh token
	rawRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		h.redirectWithAuthError(w, r, authErrServerUnavailable, "Failed to generate refresh token", err, "userID", user.ID)
		return
	}
	hashedRefreshToken := utils.HashRefreshToken(rawRefreshToken)
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // Match cookie expiry of 30 days

	if err := h.Service.CreateSession(user.ID, hashedRefreshToken, expiresAt, r.UserAgent()); err != nil {
		h.redirectWithAuthError(w, r, authErrServerUnavailable, "Failed to create session", err, "userID", user.ID)
		return
	}

	http.SetCookie(w, h.Cookies.NewSessionCookie(tok))
	http.SetCookie(w, h.Cookies.NewRefreshCookie(rawRefreshToken))

	http.Redirect(w, r, os.Getenv("CLIENT_URL"), http.StatusTemporaryRedirect)
}
