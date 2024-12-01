package service

import (
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/repository"
	"gorm.io/gorm"
)

type Service interface {
	Create(user domain.CreateUserRequest) (domain.CreateUserResponse, error)
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

func (s service) Create(user domain.CreateUserRequest) (domain.CreateUserResponse, error) {
	createUserDto := repository.CreateUserRequest{
		Id:           user.Id,
		Email:        user.Email,
		Picture:      user.Picture,
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
		TokenExpiry:  user.TokenExpiry,
	}
	id, err := s.repository.Create(createUserDto)

	if err != nil {
		return domain.CreateUserResponse{}, err
	}
	return domain.CreateUserResponse{
		Id: id,
	}, nil
}
