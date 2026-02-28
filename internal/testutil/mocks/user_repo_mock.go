package mocks

import "github.com/nickquirk/life-dashboard-server/internal/domain"

// MockUserRepository implements repository.UserRepository with function fields.
type MockUserRepository struct {
	CreateFunc                    func(domain.CreateUserRequest) (domain.CreateUserResponse, error)
	GetFunc                       func(domain.GetUserRequest) (domain.GetUserResponse, error)
	GetUsersWithRefreshTokensFunc func() ([]uint, error)
	UpdateAppRefreshTokenFunc     func(userID uint, hashedToken string) error
	GetAppRefreshTokenFunc        func(userID uint) (string, error)
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

func (m *MockUserRepository) GetUsersWithRefreshTokens() ([]uint, error) {
	if m.GetUsersWithRefreshTokensFunc != nil {
		return m.GetUsersWithRefreshTokensFunc()
	}
	return nil, nil
}

func (m *MockUserRepository) UpdateAppRefreshToken(userID uint, hashedToken string) error {
	if m.UpdateAppRefreshTokenFunc != nil {
		return m.UpdateAppRefreshTokenFunc(userID, hashedToken)
	}
	return nil
}

func (m *MockUserRepository) GetAppRefreshToken(userID uint) (string, error) {
	if m.GetAppRefreshTokenFunc != nil {
		return m.GetAppRefreshTokenFunc(userID)
	}
	return "", nil
}

func (m *MockUserRepository) DeleteUserAndData(userID uint) error {
	if m.DeleteUserAndDataFunc != nil {
		return m.DeleteUserAndDataFunc(userID)
	}
	return nil
}
