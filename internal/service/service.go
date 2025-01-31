package service

import (
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/repository"
	"gorm.io/gorm"
)

type Service interface {
	CreateUser(user domain.CreateUserRequest) (domain.CreateUserResponse, error)
	GetUser(domain.GetUserRequest) (domain.GetUserResponse, error)
}

type service struct {
	repository repository.UserRepository
}

// service handler
func NewService(db *gorm.DB) Service {
	return &service{
		repository: repository.GormUserRepository{
			Db: db,
		},
	}
}

func (s service) CreateUser(user domain.CreateUserRequest) (domain.CreateUserResponse, error) {
	resp, err := s.repository.Create(user)
	if err != nil {
		return domain.CreateUserResponse{}, err
	}
	return resp, nil
}

func (s service) GetUser(user domain.GetUserRequest) (domain.GetUserResponse, error) {
	resp, err := s.repository.Get(user)
	if err != nil {
		return domain.GetUserResponse{}, err
	}
	return resp, nil
}
