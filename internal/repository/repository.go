package repository

import (
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

type GormUserRepository struct {
	Db *gorm.DB
}

type UserRepository interface {
	CreateUser(CreateUserRequest) (uint, error)
}

func (g GormUserRepository) CreateUser(c CreateUserRequest) (uint, error) {
	user := domain.User{
		Email:        c.Email,
		Picture:      c.Picture,
		AccessToken:  c.AccessToken,
		RefreshToken: c.RefreshToken,
		TokenExpiry:  c.TokenExpiry,
	}
	result := g.Db.Create(&user)

	if result.Error != nil {
		return 0, result.Error
	}
	return user.ID, nil
}
