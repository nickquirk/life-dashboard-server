package repository

import (
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

type GormUserRepository struct {
	Db *gorm.DB
}

type UserRepository interface {
	Create(domain.CreateUserRequest) (domain.CreateUserResponse, error)
	Get(domain.GetUserRequest) (domain.GetUserResponse, error)
	GetUsersWithRefreshTokens() ([]uint, error)
}

func (r GormUserRepository) Create(c domain.CreateUserRequest) (domain.CreateUserResponse, error) {
	var user domain.User

	// 1. Check if the user already exists by Email
	err := r.Db.Where("email = ?", c.Email).First(&user).Error

	if err == nil {
		// --- USER EXISTS: UPDATE ---

		// Update standard fields
		user.AccessToken = c.AccessToken
		user.TokenExpiry = c.TokenExpiry
		user.Picture = c.Picture

		// IMPORTANT: Only update Refresh Token if Google sent a new one.
		// (Sometimes Google returns an empty string for refresh_token on re-login
		// if the user didn't manually revoke access or use prompt=consent)
		if c.RefreshToken != "" {
			user.RefreshToken = c.RefreshToken
		}

		// Save changes
		if err := r.Db.Save(&user).Error; err != nil {
			return domain.CreateUserResponse{}, err
		}

		// Return the Existing ID
		return domain.CreateUserResponse{Id: user.ID}, nil
	}

	// --- USER NEW: CREATE ---
	newUser := domain.User{
		Email:        c.Email,
		Picture:      c.Picture,
		AccessToken:  c.AccessToken,
		RefreshToken: c.RefreshToken,
		TokenExpiry:  c.TokenExpiry,
	}

	result := r.Db.Create(&newUser)
	if result.Error != nil {
		return domain.CreateUserResponse{}, result.Error
	}

	return domain.CreateUserResponse{
		Id: newUser.ID,
	}, nil
}

func (r GormUserRepository) Get(g domain.GetUserRequest) (domain.GetUserResponse, error) {
	var user domain.User
	// Find user by ID
	result := r.Db.First(&user, g.Id)

	if result.Error != nil {
		return domain.GetUserResponse{}, result.Error
	}

	return domain.GetUserResponse{
		Email:        user.Email,
		Picture:      user.Picture,
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
		TokenExpiry:  user.TokenExpiry,
	}, nil
}

func (r GormUserRepository) GetUsersWithRefreshTokens() ([]uint, error) {
	var userIDs []uint

	// Query the "users" table for non-empty refresh tokens
	err := r.Db.Table("users").
		Where("refresh_token IS NOT NULL AND refresh_token != ''").
		Pluck("id", &userIDs).Error

	if err != nil {
		return nil, err
	}

	return userIDs, nil
}
