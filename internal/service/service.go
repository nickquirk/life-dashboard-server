package service

import (
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/repository"
	"gorm.io/gorm"
)

type Service interface {
	CreateUser(user domain.CreateUserRequest) (domain.CreateUserResponse, error)
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
	createUserDto := repository.CreateUserRequest{
		Email:        user.Email,
		Picture:      user.Picture,
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
		TokenExpiry:  user.TokenExpiry,
	}
	id, err := s.repository.CreateUser(createUserDto)

	if err != nil {
		return domain.CreateUserResponse{}, err
	}
	return domain.CreateUserResponse{
		Id: id,
	}, nil
}
