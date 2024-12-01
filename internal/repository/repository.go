package repository

import (
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

type GormUserRepository struct {
	Db *gorm.DB
}

type UserRepository interface {
	Create(CreateUserRequest) (uint, error)
}

func (g GormUserRepository) Create(c CreateUserRequest) (uint, error) {

	// convert TokenExpiry from string to time.Time

	expiryLayout := "2006-01-02 15:04:05.999999999 -0700 MST m=+3669.687948501"
	expiryTime, err := time.Parse(expiryLayout, c.TokenExpiry)
	if err != nil {
		return 0, err
	}

	user := domain.User{
		Email:        c.Email,
		Picture:      c.Picture,
		AccessToken:  c.AccessToken,
		RefreshToken: c.RefreshToken,
		TokenExpiry:  expiryTime,
	}
	result := g.Db.Create(&user)

	if result.Error != nil {
		return 0, result.Error
	}
	return user.ID, nil
}
