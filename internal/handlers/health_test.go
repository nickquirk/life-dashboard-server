package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
)

func testHandler(svc *mocks.MockService) *Handler {
	return &Handler{
		Service: svc,
		Cookies: CookieConfig{
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		},
	}
}

func TestHealth_Returns200(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	h.health(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"status": "ok"`)
}

func TestReady_Returns200_WhenPingSucceeds(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()
	h.ready(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"status": "ok"`)
}

func TestReady_Returns503_WhenPingFails(t *testing.T) {
	h := testHandler(&mocks.MockService{
		PingFunc: func() error { return errors.New("db down") },
	})
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()
	h.ready(rr, r)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), `"status": "unavailable"`)
}
