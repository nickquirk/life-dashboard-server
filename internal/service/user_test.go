package service

import (
	"errors"
	"testing"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(userRepo *mocks.MockUserRepository) Service {
	return NewServiceWithRepos(userRepo, &mocks.MockTaskRepository{}, nil, nil, nil, nil, nil, nil)
}

// --- CreateUser ---

func TestCreateUser_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{
		CreateFunc: func(req domain.CreateUserRequest) (domain.CreateUserResponse, error) {
			return domain.CreateUserResponse{ID: 1}, nil
		},
	}
	svc := newTestService(repo)

	resp, err := svc.CreateUser(domain.CreateUserRequest{Email: "a@b.com"})
	require.NoError(t, err)
	assert.Equal(t, uint(1), resp.ID)
}

func TestCreateUser_RepoError(t *testing.T) {
	repo := &mocks.MockUserRepository{
		CreateFunc: func(req domain.CreateUserRequest) (domain.CreateUserResponse, error) {
			return domain.CreateUserResponse{}, errors.New("db error")
		},
	}
	svc := newTestService(repo)

	_, err := svc.CreateUser(domain.CreateUserRequest{Email: "a@b.com"})
	assert.Error(t, err)
}

// --- GetUser ---

func TestGetUser_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{
		GetFunc: func(req domain.GetUserRequest) (domain.GetUserResponse, error) {
			return domain.GetUserResponse{Email: "alice@test.com", Picture: "pic.jpg"}, nil
		},
	}
	svc := newTestService(repo)

	resp, err := svc.GetUser(domain.GetUserRequest{ID: 1})
	require.NoError(t, err)
	assert.Equal(t, "alice@test.com", resp.Email)
}

func TestGetUser_RepoError(t *testing.T) {
	repo := &mocks.MockUserRepository{
		GetFunc: func(req domain.GetUserRequest) (domain.GetUserResponse, error) {
			return domain.GetUserResponse{}, errors.New("not found")
		},
	}
	svc := newTestService(repo)

	_, err := svc.GetUser(domain.GetUserRequest{ID: 999})
	assert.Error(t, err)
}

// --- CreateSession ---

func TestCreateSession_Success(t *testing.T) {
	expiry := time.Now().Add(30 * 24 * time.Hour)
	repo := &mocks.MockUserRepository{
		CreateSessionFunc: func(userID uint, hash string, expiresAt time.Time, deviceInfo string) error {
			assert.Equal(t, uint(1), userID)
			assert.Equal(t, "hashed-token", hash)
			assert.Equal(t, expiry, expiresAt)
			assert.Equal(t, "Chrome on Mac", deviceInfo)
			return nil
		},
	}
	svc := newTestService(repo)

	err := svc.CreateSession(1, "hashed-token", expiry, "Chrome on Mac")
	assert.NoError(t, err)
}

func TestCreateSession_Error(t *testing.T) {
	repo := &mocks.MockUserRepository{
		CreateSessionFunc: func(userID uint, hash string, expiresAt time.Time, deviceInfo string) error {
			return errors.New("db error")
		},
	}
	svc := newTestService(repo)

	err := svc.CreateSession(1, "hash", time.Now(), "")
	assert.Error(t, err)
}

// --- DeleteAccount ---

func TestDeleteAccount_Success(t *testing.T) {
	called := false
	repo := &mocks.MockUserRepository{
		DeleteUserAndDataFunc: func(userID uint) error {
			called = true
			assert.Equal(t, uint(1), userID)
			return nil
		},
	}
	svc := newTestService(repo)

	err := svc.DeleteAccount(1)
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestDeleteAccount_RepoError(t *testing.T) {
	repo := &mocks.MockUserRepository{
		DeleteUserAndDataFunc: func(userID uint) error {
			return errors.New("delete failed")
		},
	}
	svc := newTestService(repo)

	err := svc.DeleteAccount(1)
	assert.Error(t, err)
	assert.Equal(t, "delete failed", err.Error())
}

// --- ValidateSession ---

func TestValidateSession_Valid(t *testing.T) {
	repo := &mocks.MockUserRepository{
		ValidateSessionFunc: func(userID uint, hash string) (bool, error) {
			assert.Equal(t, uint(1), userID)
			assert.Equal(t, "stored-hash", hash)
			return true, nil
		},
	}
	svc := newTestService(repo)

	ok, err := svc.ValidateSession(1, "stored-hash")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestValidateSession_Invalid(t *testing.T) {
	repo := &mocks.MockUserRepository{
		ValidateSessionFunc: func(userID uint, hash string) (bool, error) {
			return false, nil
		},
	}
	svc := newTestService(repo)

	ok, err := svc.ValidateSession(1, "stored-hash")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestValidateSession_Error(t *testing.T) {
	repo := &mocks.MockUserRepository{
		ValidateSessionFunc: func(userID uint, hash string) (bool, error) {
			return false, errors.New("db error")
		},
	}
	svc := newTestService(repo)

	_, err := svc.ValidateSession(1, "hash")
	assert.Error(t, err)
}

// --- DeleteSession ---

func TestDeleteSession_Success(t *testing.T) {
	called := false
	repo := &mocks.MockUserRepository{
		DeleteSessionFunc: func(userID uint, hash string) error {
			called = true
			assert.Equal(t, uint(1), userID)
			assert.Equal(t, "hash", hash)
			return nil
		},
	}
	svc := newTestService(repo)

	err := svc.DeleteSession(1, "hash")
	require.NoError(t, err)
	assert.True(t, called)
}

func TestDeleteSession_Error(t *testing.T) {
	repo := &mocks.MockUserRepository{
		DeleteSessionFunc: func(userID uint, hash string) error {
			return errors.New("db error")
		},
	}
	svc := newTestService(repo)

	err := svc.DeleteSession(1, "hash")
	assert.Error(t, err)
}
