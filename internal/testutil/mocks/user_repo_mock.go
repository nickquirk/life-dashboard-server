package mocks

import (
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

// MockUserRepository implements repository.UserRepository with function fields.
type MockUserRepository struct {
	CreateFunc                    func(domain.CreateUserRequest) (domain.CreateUserResponse, error)
	GetFunc                       func(domain.GetUserRequest) (domain.GetUserResponse, error)
	GetEmailFunc                  func(userID uint) (string, error)
	GetUsersWithRefreshTokensFunc func() ([]uint, error)
	CreateSessionFunc             func(userID uint, hashedToken string, expiresAt time.Time, deviceInfo string) error
	ValidateSessionFunc           func(userID uint, hashedToken string) (bool, error)
	DeleteSessionFunc             func(userID uint, hashedToken string) error
	DeleteUserAndDataFunc         func(userID uint) error
}

func (m *MockUserRepository) Create(req domain.CreateUserRequest) (domain.CreateUserResponse, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(req)
	}
	return domain.CreateUserResponse{}, nil
}

func (m *MockUserRepository) Get(req domain.GetUserRequest) (domain.GetUserResponse, error) {
	if m.GetFunc != nil {
		return m.GetFunc(req)
	}
	return domain.GetUserResponse{}, nil
}

func (m *MockUserRepository) GetEmail(userID uint) (string, error) {
	if m.GetEmailFunc != nil {
		return m.GetEmailFunc(userID)
	}
	return "", nil
}

func (m *MockUserRepository) GetUsersWithRefreshTokens() ([]uint, error) {
	if m.GetUsersWithRefreshTokensFunc != nil {
		return m.GetUsersWithRefreshTokensFunc()
	}
	return nil, nil
}

func (m *MockUserRepository) CreateSession(userID uint, hashedToken string, expiresAt time.Time, deviceInfo string) error {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(userID, hashedToken, expiresAt, deviceInfo)
	}
	return nil
}

func (m *MockUserRepository) ValidateSession(userID uint, hashedToken string) (bool, error) {
	if m.ValidateSessionFunc != nil {
		return m.ValidateSessionFunc(userID, hashedToken)
	}
	return false, nil
}

func (m *MockUserRepository) DeleteSession(userID uint, hashedToken string) error {
	if m.DeleteSessionFunc != nil {
		return m.DeleteSessionFunc(userID, hashedToken)
	}
	return nil
}

func (m *MockUserRepository) DeleteUserAndData(userID uint) error {
	if m.DeleteUserAndDataFunc != nil {
		return m.DeleteUserAndDataFunc(userID)
	}
	return nil
}
