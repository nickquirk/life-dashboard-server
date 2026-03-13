package service

import (
	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func (s service) CreateUser(user domain.CreateUserRequest) (domain.CreateUserResponse, error) {
	resp, err := s.userRepo.Create(user)
	if err != nil {
		return domain.CreateUserResponse{}, err
	}
	return resp, nil
}

func (s service) GetUser(user domain.GetUserRequest) (domain.GetUserResponse, error) {
	resp, err := s.userRepo.Get(user)
	if err != nil {
		return domain.GetUserResponse{}, err
	}
	return resp, nil
}

func (s service) GetUserEmail(userID uint) (string, error) {
	return s.userRepo.GetEmail(userID)
}

func (s service) UpdateAppRefreshToken(userID uint, hashedToken string) error {
	return s.userRepo.UpdateAppRefreshToken(userID, hashedToken)
}

func (s service) GetAppRefreshToken(userID uint) (string, error) {
	return s.userRepo.GetAppRefreshToken(userID)
}

func (s service) DeleteAccount(userID uint) error {
	return s.userRepo.DeleteUserAndData(userID)
}
