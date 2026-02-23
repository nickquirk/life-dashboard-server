package handlers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCookieConfig_DevDefaults(t *testing.T) {
	// No env vars set → dev mode
	t.Setenv("ENV", "")
	t.Setenv("CROSS_ORIGIN", "")
	t.Setenv("COOKIE_DOMAIN", "")

	cc := NewCookieConfig()
	assert.False(t, cc.Secure)
	assert.Equal(t, http.SameSiteLaxMode, cc.SameSite)
	assert.Empty(t, cc.Domain)
}

func TestNewCookieConfig_Prod(t *testing.T) {
	t.Setenv("ENV", "prod")
	t.Setenv("CROSS_ORIGIN", "")
	t.Setenv("COOKIE_DOMAIN", "")

	cc := NewCookieConfig()
	assert.True(t, cc.Secure)
	assert.Equal(t, http.SameSiteLaxMode, cc.SameSite)
}

func TestNewCookieConfig_CrossOrigin(t *testing.T) {
	t.Setenv("ENV", "")
	t.Setenv("CROSS_ORIGIN", "true")
	t.Setenv("COOKIE_DOMAIN", ".example.com")

	cc := NewCookieConfig()
	assert.True(t, cc.Secure)
	assert.Equal(t, http.SameSiteNoneMode, cc.SameSite)
	assert.Equal(t, ".example.com", cc.Domain)
}

func TestSessionCookie(t *testing.T) {
	cc := CookieConfig{Secure: true, SameSite: http.SameSiteNoneMode, Domain: ".example.com"}
	c := cc.NewSessionCookie("jwt-token-value")

	assert.Equal(t, "life-dashboard", c.Name)
	assert.Equal(t, "jwt-token-value", c.Value)
	assert.Equal(t, 6*60*60, c.MaxAge)
	assert.True(t, c.HttpOnly)
	assert.True(t, c.Secure)
	assert.Equal(t, http.SameSiteNoneMode, c.SameSite)
	assert.Equal(t, ".example.com", c.Domain)
}

func TestRefreshCookie(t *testing.T) {
	cc := CookieConfig{Secure: true, SameSite: http.SameSiteLaxMode}
	c := cc.NewRefreshCookie("refresh-token-value")

	assert.Equal(t, "life-dashboard-refresh", c.Name)
	assert.Equal(t, "refresh-token-value", c.Value)
	assert.Equal(t, 30*24*60*60, c.MaxAge)
	assert.True(t, c.HttpOnly)
}

func TestExpireSessionCookie(t *testing.T) {
	cc := CookieConfig{}
	c := cc.ExpireSessionCookie()

	assert.Equal(t, "life-dashboard", c.Name)
	assert.Empty(t, c.Value)
	assert.Equal(t, -1, c.MaxAge)
}

func TestOAuthStateCookie(t *testing.T) {
	cc := CookieConfig{Secure: true, SameSite: http.SameSiteNoneMode}
	c := cc.NewOAuthStateCookie("random-state")

	assert.Equal(t, "oauthstate", c.Name)
	assert.Equal(t, "random-state", c.Value)
	assert.Equal(t, 10*60, c.MaxAge)
	assert.True(t, c.HttpOnly)
}
