package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nickquirk/life-dashboard-server/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticate_ValidCookie(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests-32chars!")

	token, err := utils.GenerateToken(42, "alice@example.com")
	require.NoError(t, err)

	var capturedUserID uint
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := r.Context().Value(UserIDKey).(uint)
		if ok {
			capturedUserID = uid
		}
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	r.AddCookie(&http.Cookie{Name: "life-dashboard", Value: token})
	rr := httptest.NewRecorder()

	Authenticate(inner).ServeHTTP(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, uint(42), capturedUserID)
}

func TestAuthenticate_MissingCookie(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests-32chars!")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	r := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rr := httptest.NewRecorder()

	Authenticate(inner).ServeHTTP(rr, r)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests-32chars!")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	r := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	r.AddCookie(&http.Cookie{Name: "life-dashboard", Value: "garbage-token"})
	rr := httptest.NewRecorder()

	Authenticate(inner).ServeHTTP(rr, r)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticateOIDC_NoAudience_Passthrough(t *testing.T) {
	t.Setenv("CLOUD_RUN_URL", "")

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodPost, "/api/jobs/sync/trigger", nil)
	rr := httptest.NewRecorder()

	AuthenticateOIDC(inner).ServeHTTP(rr, r)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rr.Code)
}
