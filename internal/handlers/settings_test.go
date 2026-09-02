package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
)

// --- getUserSettings ---

func TestGetUserSettings_Success(t *testing.T) {
	svc := &mocks.MockService{
		GetUserSettingsFunc: func(req domain.GetUserSettingsRequest) (domain.UserSettingsResponse, error) {
			assert.Equal(t, uint(1), req.UserID)
			return domain.UserSettingsResponse{Settings: domain.Settings{Version: 1}}, nil
		},
	}
	h := testHandler(svc)
	r := withUser(httptest.NewRequest(http.MethodGet, "/api/settings", nil), 1)
	rr := httptest.NewRecorder()

	h.getUserSettings(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"version":1`)
}

func TestGetUserSettings_NoUser(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rr := httptest.NewRecorder()

	h.getUserSettings(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestGetUserSettings_ServiceError(t *testing.T) {
	svc := &mocks.MockService{
		GetUserSettingsFunc: func(req domain.GetUserSettingsRequest) (domain.UserSettingsResponse, error) {
			return domain.UserSettingsResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	r := withUser(httptest.NewRequest(http.MethodGet, "/api/settings", nil), 1)
	rr := httptest.NewRecorder()

	h.getUserSettings(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- updateUserSettings ---

func TestUpdateUserSettings_Success(t *testing.T) {
	patch := []byte(`{}`)
	svc := &mocks.MockService{
		UpdateUserSettingsFunc: func(req domain.UpdateUserSettingsRequest) (domain.UserSettingsResponse, error) {
			// The user ID must come from the auth context, never the body
			assert.Equal(t, uint(1), req.UserID)
			assert.Equal(t, patch, req.Patch)
			return domain.UserSettingsResponse{Settings: domain.Settings{Version: 1}}, nil
		},
	}
	h := testHandler(svc)
	r := withUser(httptest.NewRequest(http.MethodPatch, "/api/settings", bytes.NewReader(patch)), 1)
	rr := httptest.NewRecorder()

	h.updateUserSettings(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"version":1`)
}

func TestUpdateUserSettings_PatchBodyIsNotDecodedByHandler(t *testing.T) {
	patch := []byte(`{"anything":true}`)
	svc := &mocks.MockService{
		UpdateUserSettingsFunc: func(req domain.UpdateUserSettingsRequest) (domain.UserSettingsResponse, error) {
			assert.Equal(t, patch, req.Patch)
			return domain.UserSettingsResponse{Settings: domain.DefaultSettings()}, nil
		},
	}
	h := testHandler(svc)
	r := withUser(httptest.NewRequest(http.MethodPatch, "/api/settings", bytes.NewReader(patch)), 1)
	rr := httptest.NewRecorder()

	h.updateUserSettings(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestUpdateUserSettings_InvalidInput(t *testing.T) {
	svc := &mocks.MockService{
		UpdateUserSettingsFunc: func(req domain.UpdateUserSettingsRequest) (domain.UserSettingsResponse, error) {
			return domain.UserSettingsResponse{}, fmt.Errorf("%w: bad", domain.ErrInvalidInput)
		},
	}
	h := testHandler(svc)
	r := withUser(httptest.NewRequest(http.MethodPatch, "/api/settings", bytes.NewReader([]byte(`{}`))), 1)
	rr := httptest.NewRecorder()

	h.updateUserSettings(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateUserSettings_ServiceError(t *testing.T) {
	svc := &mocks.MockService{
		UpdateUserSettingsFunc: func(req domain.UpdateUserSettingsRequest) (domain.UserSettingsResponse, error) {
			return domain.UserSettingsResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	r := withUser(httptest.NewRequest(http.MethodPatch, "/api/settings", bytes.NewReader([]byte(`{}`))), 1)
	rr := httptest.NewRecorder()

	h.updateUserSettings(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestUpdateUserSettings_NoUser(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPatch, "/api/settings", nil)
	rr := httptest.NewRecorder()

	h.updateUserSettings(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestUpdateUserSettings_BodyTooLarge(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	body := bytes.Repeat([]byte("a"), maxSettingsBodyBytes+1)
	r := withUser(httptest.NewRequest(http.MethodPatch, "/api/settings", bytes.NewReader(body)), 1)
	rr := httptest.NewRecorder()

	h.updateUserSettings(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
