package testutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/handlers"
	"github.com/nickquirk/life-dashboard-server/internal/service"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
)

// TestCookieConfig returns a CookieConfig suitable for tests (insecure, Lax).
func TestCookieConfig() handlers.CookieConfig {
	return handlers.CookieConfig{
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Domain:   "",
	}
}

// NewTestHandler creates a Handler with the given mock service and test cookie config.
func NewTestHandler(svc service.Service) *handlers.Handler {
	return &handlers.Handler{
		Service: svc,
		Cookies: TestCookieConfig(),
	}
}

// WithUserID returns a new request whose context carries the given userID.
func WithUserID(r *http.Request, userID uint) *http.Request {
	ctx := context.WithValue(r.Context(), handlers.UserIDKey, userID)
	return r.WithContext(ctx)
}

// WithChiURLParam returns a new request with chi URL params set.
func WithChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// WithChiURLParams returns a new request with multiple chi URL params set.
func WithChiURLParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// WithUserAndChiParam sets both userID in context and a chi URL param.
func WithUserAndChiParam(r *http.Request, userID uint, paramKey, paramValue string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(paramKey, paramValue)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, handlers.UserIDKey, userID)
	return r.WithContext(ctx)
}

// SetupJWTEnv sets the JWT_SECRET env var for tests that need JWT operations.
func SetupJWTEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests-32chars!")
}

// NewMockService creates a fresh MockService.
func NewMockService() *mocks.MockService {
	return &mocks.MockService{}
}

// ExecuteRequest is a convenience helper to execute a handler and return the recorder.
func ExecuteRequest(handler http.HandlerFunc, r *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, r)
	return rr
}
