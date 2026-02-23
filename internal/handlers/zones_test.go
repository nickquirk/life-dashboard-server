package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// --- createZone ---

func TestCreateZone_Success(t *testing.T) {
	svc := &mocks.MockService{
		CreateZoneFunc: func(req domain.CreateZoneRequest) (domain.CreateZoneResponse, error) {
			return domain.CreateZoneResponse{ID: 1}, nil
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.CreateZoneRequest{Label: "Work", StartTime: "09:00", EndTime: "17:00"})
	r := httptest.NewRequest(http.MethodPost, "/api/zones", bytes.NewReader(body))
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.createZone(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"id":1`)
}

func TestCreateZone_BadBody(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPost, "/api/zones", bytes.NewReader([]byte("bad")))
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.createZone(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateZone_NoUser(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPost, "/api/zones", nil)
	rr := httptest.NewRecorder()

	h.createZone(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestCreateZone_ServiceError(t *testing.T) {
	svc := &mocks.MockService{
		CreateZoneFunc: func(req domain.CreateZoneRequest) (domain.CreateZoneResponse, error) {
			return domain.CreateZoneResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.CreateZoneRequest{Label: "Work"})
	r := httptest.NewRequest(http.MethodPost, "/api/zones", bytes.NewReader(body))
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.createZone(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- getZones ---

func TestGetZones_Success(t *testing.T) {
	svc := &mocks.MockService{
		GetZonesFunc: func(req domain.GetZonesRequest) (domain.GetZonesResponse, error) {
			return domain.GetZonesResponse{
				Zones: []domain.ZoneResponse{{ID: 1, Label: "Work"}},
			}, nil
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/api/zones", nil)
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.getZones(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Work")
}

func TestGetZones_Error(t *testing.T) {
	svc := &mocks.MockService{
		GetZonesFunc: func(req domain.GetZonesRequest) (domain.GetZonesResponse, error) {
			return domain.GetZonesResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/api/zones", nil)
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.getZones(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- updateZone ---

func TestUpdateZone_Success(t *testing.T) {
	label := "Updated Work"
	svc := &mocks.MockService{
		UpdateZoneFunc: func(req domain.UpdateZoneRequest) (domain.UpdateZoneResponse, error) {
			return domain.UpdateZoneResponse{ID: 5, Label: label}, nil
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.UpdateZoneRequest{Label: &label})
	r := httptest.NewRequest(http.MethodPatch, "/api/zones/5", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.updateZone(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Updated Work")
}

func TestUpdateZone_InvalidID(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPatch, "/api/zones/abc", nil)
	r = withUserAndChi(r, 1, "id", "abc")
	rr := httptest.NewRecorder()

	h.updateZone(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateZone_BadBody(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPatch, "/api/zones/5", bytes.NewReader([]byte("bad")))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.updateZone(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateZone_Error(t *testing.T) {
	label := "Fail"
	svc := &mocks.MockService{
		UpdateZoneFunc: func(req domain.UpdateZoneRequest) (domain.UpdateZoneResponse, error) {
			return domain.UpdateZoneResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.UpdateZoneRequest{Label: &label})
	r := httptest.NewRequest(http.MethodPatch, "/api/zones/5", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.updateZone(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- deleteZone ---

func TestDeleteZone_Success(t *testing.T) {
	svc := &mocks.MockService{
		DeleteZoneFunc: func(req domain.DeleteZonesRequest) (domain.DeleteZonesResponse, error) {
			return domain.DeleteZonesResponse{Message: "Zone deleted successfully"}, nil
		},
	}
	h := testHandler(svc)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "10")
	r := httptest.NewRequest(http.MethodDelete, "/api/zones/10", nil)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, UserIDKey, uint(1))
	r = r.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.deleteZone(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "deleted")
}

func TestDeleteZone_InvalidID(t *testing.T) {
	h := testHandler(&mocks.MockService{})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc")
	r := httptest.NewRequest(http.MethodDelete, "/api/zones/abc", nil)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, UserIDKey, uint(1))
	r = r.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.deleteZone(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeleteZone_NotFound(t *testing.T) {
	svc := &mocks.MockService{
		DeleteZoneFunc: func(req domain.DeleteZonesRequest) (domain.DeleteZonesResponse, error) {
			return domain.DeleteZonesResponse{}, gorm.ErrRecordNotFound
		},
	}
	h := testHandler(svc)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
	r := httptest.NewRequest(http.MethodDelete, "/api/zones/999", nil)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, UserIDKey, uint(1))
	r = r.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.deleteZone(rr, r)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDeleteZone_Error(t *testing.T) {
	svc := &mocks.MockService{
		DeleteZoneFunc: func(req domain.DeleteZonesRequest) (domain.DeleteZonesResponse, error) {
			return domain.DeleteZonesResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "10")
	r := httptest.NewRequest(http.MethodDelete, "/api/zones/10", nil)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, UserIDKey, uint(1))
	r = r.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.deleteZone(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
