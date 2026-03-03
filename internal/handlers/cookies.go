package handlers

import (
	"net/http"
	"os"
)

// CookieConfig holds the resolved cookie settings for the current environment.
// Initialised once at startup via NewCookieConfig().
type CookieConfig struct {
	Secure   bool
	SameSite http.SameSite
	Domain   string // Optional: set via COOKIE_DOMAIN env var
}

// NewCookieConfig builds cookie settings based on the current environment.
//
// Cross-origin deployments (e.g. separate Cloud Run services for FE and BE)
// require SameSite=None + Secure=true so the browser actually sends cookies
// on cross-origin requests.
//
// Environment variables:
//   - ENV=prod           → Secure=true  (required for HTTPS)
//   - CROSS_ORIGIN=true  → SameSite=None (required when FE/BE are different origins)
//   - COOKIE_DOMAIN      → optional, e.g. ".example.com" for shared subdomain cookies
func NewCookieConfig() CookieConfig {
	isProd := os.Getenv("ENV") == "prod"
	isCrossOrigin := os.Getenv("CROSS_ORIGIN") == "true"

	sameSite := http.SameSiteLaxMode
	if isCrossOrigin {
		// SameSite=None is mandatory for cross-origin cookie delivery.
		// Browsers also require Secure=true when SameSite=None.
		sameSite = http.SameSiteNoneMode
	}

	return CookieConfig{
		Secure:   isProd || isCrossOrigin, // SameSite=None requires Secure even in dev (use HTTPS proxy)
		SameSite: sameSite,
		Domain:   os.Getenv("COOKIE_DOMAIN"),
	}
}

// NewSessionCookie creates the main auth cookie ("life-dashboard").
func (cc CookieConfig) NewSessionCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     "life-dashboard",
		Value:    token,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 Days
		HttpOnly: true,
		Secure:   cc.Secure,
		SameSite: cc.SameSite,
		Domain:   cc.Domain,
	}
}

// ExpireSessionCookie returns a cookie that tells the browser to delete the session.
func (cc CookieConfig) ExpireSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     "life-dashboard",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cc.Secure,
		SameSite: cc.SameSite,
		Domain:   cc.Domain,
	}
}

// NewOAuthStateCookie creates the short-lived CSRF state cookie for the Google login flow.
func (cc CookieConfig) NewOAuthStateCookie(state string) *http.Cookie {
	return &http.Cookie{
		Name:     "oauthstate",
		Value:    state,
		Path:     "/",
		MaxAge:   10 * 60, // 10 minutes
		HttpOnly: true,
		Secure:   cc.Secure,
		SameSite: cc.SameSite,
		Domain:   cc.Domain,
	}
}

// ExpireOAuthStateCookie clears the state cookie after it's been validated.
func (cc CookieConfig) ExpireOAuthStateCookie() *http.Cookie {
	return &http.Cookie{
		Name:     "oauthstate",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cc.Secure,
		SameSite: cc.SameSite,
		Domain:   cc.Domain,
	}
}

// NewRefreshCookie creates the refresh token cookie ("life-dashboard-refresh").
func (cc CookieConfig) NewRefreshCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     "life-dashboard-refresh",
		Value:    token,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HttpOnly: true,
		Secure:   cc.Secure,
		SameSite: cc.SameSite,
		Domain:   cc.Domain,
	}
}

// ExpireRefreshCookie returns a cookie that tells the browser to delete the refresh token.
func (cc CookieConfig) ExpireRefreshCookie() *http.Cookie {
	return &http.Cookie{
		Name:     "life-dashboard-refresh",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cc.Secure,
		SameSite: cc.SameSite,
		Domain:   cc.Domain,
	}
}
