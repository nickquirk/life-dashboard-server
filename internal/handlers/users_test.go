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
