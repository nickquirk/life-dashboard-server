package repository

import (
	"fmt"

	"github.com/nickquirk/life-dashboard-server/internal/crypto"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

type GormUserRepository struct {
	Db        *gorm.DB
	Encryptor crypto.TokenEncryptor
}

type UserRepository interface {
	Create(domain.CreateUserRequest) (domain.CreateUserResponse, error)
	Get(domain.GetUserRequest) (domain.GetUserResponse, error)
	GetUsersWithRefreshTokens() ([]uint, error)
	UpdateAppRefreshToken(userID uint, hashedToken string) error
	GetAppRefreshToken(userID uint) (string, error)
	DeleteUserAndData(userID uint) error
}

func (r GormUserRepository) Create(c domain.CreateUserRequest) (domain.CreateUserResponse, error) {
	var user domain.User

	// Encrypt tokens before any DB write
	encAccessToken, err := r.Encryptor.Encrypt(c.AccessToken)
	if err != nil {
		return domain.CreateUserResponse{}, fmt.Errorf("failed to encrypt access token: %w", err)
	}

	// 1. Check if the user already exists by Email
	err = r.Db.Where("email = ?", c.Email).First(&user).Error

	if err == nil {
		// --- USER EXISTS: UPDATE ---

		// Update standard fields
		user.AccessToken = encAccessToken
		user.TokenExpiry = c.TokenExpiry
		user.Picture = c.Picture

		// IMPORTANT: Only update Refresh Token if Google sent a new one.
		// (Sometimes Google returns an empty string for refresh_token on re-login
		// if the user didn't manually revoke access or use prompt=consent)
		if c.RefreshToken != "" {
			encRefreshToken, err := r.Encryptor.Encrypt(c.RefreshToken)
			if err != nil {
				return domain.CreateUserResponse{}, fmt.Errorf("failed to encrypt refresh token: %w", err)
			}
			user.RefreshToken = encRefreshToken
		}

		// Save changes
		if err := r.Db.Save(&user).Error; err != nil {
			return domain.CreateUserResponse{}, err
		}

		// Return the Existing ID
		return domain.CreateUserResponse{ID: user.ID}, nil
	}

	// --- USER NEW: CREATE ---
	encRefreshToken, err := r.Encryptor.Encrypt(c.RefreshToken)
	if err != nil {
		return domain.CreateUserResponse{}, fmt.Errorf("failed to encrypt refresh token: %w", err)
	}

	newUser := domain.User{
		Email:        c.Email,
		Picture:      c.Picture,
		AccessToken:  encAccessToken,
		RefreshToken: encRefreshToken,
		TokenExpiry:  c.TokenExpiry,
	}

	result := r.Db.Create(&newUser)
	if result.Error != nil {
		return domain.CreateUserResponse{}, result.Error
	}

	return domain.CreateUserResponse{
		ID: newUser.ID,
	}, nil
}

func (r GormUserRepository) Get(g domain.GetUserRequest) (domain.GetUserResponse, error) {
	var user domain.User
	// Find user by ID
	result := r.Db.First(&user, g.ID)

	if result.Error != nil {
		return domain.GetUserResponse{}, result.Error
	}

	// Decrypt tokens after reading from DB
	accessToken, err := r.Encryptor.Decrypt(user.AccessToken)
	if err != nil {
		return domain.GetUserResponse{}, fmt.Errorf("failed to decrypt access token: %w", err)
	}
	refreshToken, err := r.Encryptor.Decrypt(user.RefreshToken)
	if err != nil {
		return domain.GetUserResponse{}, fmt.Errorf("failed to decrypt refresh token: %w", err)
	}

	return domain.GetUserResponse{
		Email:        user.Email,
		Picture:      user.Picture,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
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

func (r GormUserRepository) UpdateAppRefreshToken(userID uint, hashedToken string) error {
	return r.Db.Model(&domain.User{}).Where("id = ?", userID).Update("app_refresh_token", hashedToken).Error
}

func (r GormUserRepository) GetAppRefreshToken(userID uint) (string, error) {
	var token string
	err := r.Db.Model(&domain.User{}).Where("id = ?", userID).Pluck("app_refresh_token", &token).Error
	if err != nil {
		return "", err
	}
	return token, nil
}

func (r GormUserRepository) DeleteUserAndData(userID uint) error {
	tx := r.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Permanently delete Scratchpad entries
	if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&domain.Scratchpad{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Permanently delete Feedback
	if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&domain.Feedback{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Permanently delete Routine Instances (Child)
	if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&domain.RoutineInstance{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Permanently delete Routines (Parent)
	if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&domain.Routine{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Permanently delete Zones
	if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&domain.Zone{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Permanently delete SUBTASKS first (satisfies the self-referencing foreign key)
	if err := tx.Exec("DELETE FROM tasks WHERE parent IS NOT NULL AND task_list_id IN (SELECT id FROM task_lists WHERE user_id = ?)", userID).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Permanently delete PARENT TASKS
	if err := tx.Exec("DELETE FROM tasks WHERE task_list_id IN (SELECT id FROM task_lists WHERE user_id = ?)", userID).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Permanently delete TaskLists
	if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&domain.TaskList{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Permanently delete the User
	if err := tx.Unscoped().Where("id = ?", userID).Delete(&domain.User{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
