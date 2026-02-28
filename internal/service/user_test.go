package service

import (
	"errors"
	"testing"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestService(userRepo *mocks.MockUserRepository) Service {
	return NewServiceWithRepos(userRepo, &mocks.MockTaskRepository{}, nil, nil)
}

// --- CreateUser ---

func TestCreateUser_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{
		CreateFunc: func(req domain.CreateUserRequest) (domain.CreateUserResponse, error) {
			return domain.CreateUserResponse{Id: 1}, nil
		},
	}
	svc := newTestService(repo)

	resp, err := svc.CreateUser(domain.CreateUserRequest{Email: "a@b.com"})
	require.NoError(t, err)
	assert.Equal(t, uint(1), resp.Id)
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

	resp, err := svc.GetUser(domain.GetUserRequest{Id: 1})
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

	_, err := svc.GetUser(domain.GetUserRequest{Id: 999})
	assert.Error(t, err)
}

// --- UpdateAppRefreshToken ---

func TestUpdateAppRefreshToken_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{
		UpdateAppRefreshTokenFunc: func(userID uint, hash string) error {
			assert.Equal(t, uint(1), userID)
			assert.Equal(t, "hashed-token", hash)
			return nil
		},
	}
	svc := newTestService(repo)

	err := svc.UpdateAppRefreshToken(1, "hashed-token")
	assert.NoError(t, err)
}

func TestUpdateAppRefreshToken_Error(t *testing.T) {
	repo := &mocks.MockUserRepository{
		UpdateAppRefreshTokenFunc: func(userID uint, hash string) error {
			return errors.New("db error")
		},
	}
	svc := newTestService(repo)

	err := svc.UpdateAppRefreshToken(1, "hash")
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

// --- GetAppRefreshToken ---

func TestGetAppRefreshToken_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{
		GetAppRefreshTokenFunc: func(userID uint) (string, error) {
			return "stored-hash", nil
		},
	}
	svc := newTestService(repo)

	token, err := svc.GetAppRefreshToken(1)
	require.NoError(t, err)
	assert.Equal(t, "stored-hash", token)
}

func TestGetAppRefreshToken_Error(t *testing.T) {
	repo := &mocks.MockUserRepository{
		GetAppRefreshTokenFunc: func(userID uint) (string, error) {
			return "", errors.New("db error")
		},
	}
	svc := newTestService(repo)

	_, err := svc.GetAppRefreshToken(1)
	assert.Error(t, err)
}
