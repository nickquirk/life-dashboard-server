package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/nickquirk/life-dashboard-server/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogout_ClearsCookies(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests-32chars!")

	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rr := httptest.NewRecorder()

	h.logout(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)

	cookies := rr.Result().Cookies()
	var names []string
	for _, c := range cookies {
		names = append(names, c.Name)
		assert.Equal(t, -1, c.MaxAge, "cookie %s should be expired", c.Name)
	}
	assert.Contains(t, names, "life-dashboard")
	assert.Contains(t, names, "life-dashboard-refresh")
}

func TestRefreshToken_Success(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests-32chars!")

	rawRefresh := "test-refresh-token-value"
	providedHash := utils.HashRefreshToken(rawRefresh)

	token, err := utils.GenerateToken(1, "user@example.com")
	require.NoError(t, err)

	deleteCalled := false
	createCalled := false
	svc := &mocks.MockService{
		ValidateSessionFunc: func(userID uint, hash string) (bool, error) {
			assert.Equal(t, uint(1), userID)
			assert.Equal(t, providedHash, hash)
			return true, nil
		},
		GetUserEmailFunc: func(userID uint) (string, error) {
			return "user@example.com", nil
		},
		DeleteSessionFunc: func(userID uint, hash string) error {
			deleteCalled = true
			assert.Equal(t, providedHash, hash)
			return nil
		},
		CreateSessionFunc: func(userID uint, hash string, expiresAt time.Time, deviceInfo string) error {
			createCalled = true
			assert.NotEqual(t, providedHash, hash, "should rotate to a new hash")
			assert.True(t, expiresAt.After(time.Now()))
			return nil
		},
	}

	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	r.AddCookie(&http.Cookie{Name: "life-dashboard", Value: token})
	r.AddCookie(&http.Cookie{Name: "life-dashboard-refresh", Value: rawRefresh})
	rr := httptest.NewRecorder()

	h.refreshToken(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, deleteCalled, "old session should be deleted")
	assert.True(t, createCalled, "new session should be created")
	cookies := rr.Result().Cookies()
	assert.GreaterOrEqual(t, len(cookies), 2)
}

func TestRefreshToken_DeleteSessionFails(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests-32chars!")

	token, err := utils.GenerateToken(1, "user@example.com")
	require.NoError(t, err)

	createCalled := false
	svc := &mocks.MockService{
		ValidateSessionFunc: func(userID uint, hash string) (bool, error) { return true, nil },
		GetUserEmailFunc:    func(userID uint) (string, error) { return "user@example.com", nil },
		DeleteSessionFunc: func(userID uint, hash string) error {
			return errors.New("db error")
		},
		CreateSessionFunc: func(userID uint, hash string, expiresAt time.Time, deviceInfo string) error {
			createCalled = true
			return nil
		},
	}

	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	r.AddCookie(&http.Cookie{Name: "life-dashboard", Value: token})
	r.AddCookie(&http.Cookie{Name: "life-dashboard-refresh", Value: "raw"})
	rr := httptest.NewRecorder()

	h.refreshToken(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.False(t, createCalled, "new session must not be created if old delete failed")
}

func TestRefreshToken_MissingJWTCookie(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests-32chars!")

	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	rr := httptest.NewRecorder()

	h.refreshToken(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRefreshToken_MissingRefreshCookie(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests-32chars!")

	token, err := utils.GenerateToken(1, "user@example.com")
	require.NoError(t, err)

	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	r.AddCookie(&http.Cookie{Name: "life-dashboard", Value: token})
	rr := httptest.NewRecorder()

	h.refreshToken(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRefreshToken_InvalidSession(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests-32chars!")

	token, err := utils.GenerateToken(1, "user@example.com")
	require.NoError(t, err)

	svc := &mocks.MockService{
		ValidateSessionFunc: func(userID uint, hash string) (bool, error) {
			return false, nil
		},
	}

	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	r.AddCookie(&http.Cookie{Name: "life-dashboard", Value: token})
	r.AddCookie(&http.Cookie{Name: "life-dashboard-refresh", Value: "some-refresh-token"})
	rr := httptest.NewRecorder()

	h.refreshToken(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRefreshToken_UserNotFound(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests-32chars!")

	rawRefresh := "test-refresh-token"

	token, err := utils.GenerateToken(1, "user@example.com")
	require.NoError(t, err)

	svc := &mocks.MockService{
		ValidateSessionFunc: func(userID uint, hash string) (bool, error) {
			return true, nil
		},
		GetUserEmailFunc: func(userID uint) (string, error) {
			return "", errors.New("not found")
		},
	}

	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	r.AddCookie(&http.Cookie{Name: "life-dashboard", Value: token})
	r.AddCookie(&http.Cookie{Name: "life-dashboard-refresh", Value: rawRefresh})
	rr := httptest.NewRecorder()

	h.refreshToken(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestGetCurrentUser_Success(t *testing.T) {
	svc := &mocks.MockService{
		GetUserFunc: func(req domain.GetUserRequest) (domain.GetUserResponse, error) {
			return domain.GetUserResponse{Email: "alice@test.com", Picture: "pic.jpg"}, nil
		},
	}

	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	ctx := context.WithValue(r.Context(), UserIDKey, uint(42))
	r = r.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.getCurrentUser(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "alice@test.com")
}

func TestGetCurrentUser_NoContext(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rr := httptest.NewRecorder()

	h.getCurrentUser(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestGetCurrentUser_ServiceError(t *testing.T) {
	svc := &mocks.MockService{
		GetUserFunc: func(req domain.GetUserRequest) (domain.GetUserResponse, error) {
			return domain.GetUserResponse{}, errors.New("db error")
		},
	}

	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	ctx := context.WithValue(r.Context(), UserIDKey, uint(1))
	r = r.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.getCurrentUser(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
