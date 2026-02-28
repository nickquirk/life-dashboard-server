package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
)

func TestGetUserHTTP_Success(t *testing.T) {
	svc := &mocks.MockService{
		GetUserFunc: func(req domain.GetUserRequest) (domain.GetUserResponse, error) {
			return domain.GetUserResponse{Email: "alice@test.com", Picture: "pic.jpg"}, nil
		},
	}
	h := testHandler(svc)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "42")
	r := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetUserHTTP(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "alice@test.com")
	assert.Contains(t, rr.Body.String(), "pic.jpg")
}

func TestGetUserHTTP_InvalidID(t *testing.T) {
	h := testHandler(&mocks.MockService{})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc")
	r := httptest.NewRequest(http.MethodGet, "/api/users/abc", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetUserHTTP(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetUserHTTP_NotFound(t *testing.T) {
	svc := &mocks.MockService{
		GetUserFunc: func(req domain.GetUserRequest) (domain.GetUserResponse, error) {
			return domain.GetUserResponse{}, errors.New("record not found")
		},
	}
	h := testHandler(svc)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
	r := httptest.NewRequest(http.MethodGet, "/api/users/999", nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetUserHTTP(rr, r)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// --- DeleteAccount ---

func TestDeleteAccount_Success(t *testing.T) {
	svc := &mocks.MockService{
		DeleteAccountFunc: func(userID uint) error {
			assert.Equal(t, uint(42), userID)
			return nil
		},
	}
	h := testHandler(svc)

	r := httptest.NewRequest(http.MethodDelete, "/api/users/me", nil)
	ctx := context.WithValue(r.Context(), UserIDKey, uint(42))
	r = r.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.deleteAccount(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Account and all data deleted")

	// Verify cookies are expired
	cookies := rr.Result().Cookies()
	var names []string
	for _, c := range cookies {
		names = append(names, c.Name)
		assert.Equal(t, -1, c.MaxAge, "cookie %s should be expired", c.Name)
	}
	assert.Contains(t, names, "life-dashboard")
	assert.Contains(t, names, "life-dashboard-refresh")
}

func TestDeleteAccount_NoContext(t *testing.T) {
	h := testHandler(&mocks.MockService{})

	r := httptest.NewRequest(http.MethodDelete, "/api/users/me", nil)
	rr := httptest.NewRecorder()

	h.deleteAccount(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestDeleteAccount_ServiceError(t *testing.T) {
	svc := &mocks.MockService{
		DeleteAccountFunc: func(userID uint) error {
			return errors.New("delete failed")
		},
	}
	h := testHandler(svc)

	r := httptest.NewRequest(http.MethodDelete, "/api/users/me", nil)
	ctx := context.WithValue(r.Context(), UserIDKey, uint(1))
	r = r.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.deleteAccount(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
