package repository

import (
	"fmt"
	"time"

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
	GetEmail(userID uint) (string, error)
	GetUsersWithRefreshTokens() ([]uint, error)
	CreateSession(userID uint, hashedToken string, expiresAt time.Time, deviceInfo string) error
	ValidateSession(userID uint, hashedToken string) (bool, error)
	DeleteSession(userID uint, hashedToken string) error
	DeleteUserAndData(userID uint) error
}

// MaxSessionsPerUser caps how many active sessions a user can hold at once.
// On overflow, the session with the soonest expiry is evicted.
const MaxSessionsPerUser = 10

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

func (r GormUserRepository) GetEmail(userID uint) (string, error) {
	var email string
	err := r.Db.Model(&domain.User{}).Where("id = ?", userID).Pluck("email", &email).Error
	if err != nil {
		return "", err
	}
	return email, nil
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

// CreateSession inserts a new session, opportunistically purges expired rows for
// the same user, and evicts the oldest active session when the per-user cap is hit.
func (r GormUserRepository) CreateSession(userID uint, hashedToken string, expiresAt time.Time, deviceInfo string) error {
	return r.Db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND expires_at <= ?", userID, time.Now()).Delete(&domain.Session{}).Error; err != nil {
			return err
		}

		var activeCount int64
		if err := tx.Model(&domain.Session{}).Where("user_id = ?", userID).Count(&activeCount).Error; err != nil {
			return err
		}

		if activeCount >= int64(MaxSessionsPerUser) {
			toRemove := int(activeCount) - MaxSessionsPerUser + 1
			var oldest []domain.Session
			if err := tx.Where("user_id = ?", userID).Order("expires_at ASC").Limit(toRemove).Find(&oldest).Error; err != nil {
				return err
			}
			ids := make([]uint, len(oldest))
			for i, s := range oldest {
				ids[i] = s.ID
			}
			if len(ids) > 0 {
				if err := tx.Delete(&domain.Session{}, ids).Error; err != nil {
					return err
				}
			}
		}

		session := domain.Session{
			UserID:          userID,
			AppRefreshToken: hashedToken,
			ExpiresAt:       expiresAt,
			DeviceInfo:      deviceInfo,
		}
		return tx.Create(&session).Error
	})
}

// Find a specific session by the hashed token
func (r GormUserRepository) ValidateSession(userID uint, hashedToken string) (bool, error) {
	var count int64
	err := r.Db.Model(&domain.Session{}).
		Where("user_id = ? AND app_refresh_token = ? AND expires_at > ?", userID, hashedToken, time.Now()).
		Count(&count).Error

	return count > 0, err
}

// Delete a session (for logout or token rotation)
func (r GormUserRepository) DeleteSession(userID uint, hashedToken string) error {
	return r.Db.Where("user_id = ? AND app_refresh_token = ?", userID, hashedToken).Delete(&domain.Session{}).Error
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

	// Permanently delete Note Items (Child) — must come before Notes
	if err := tx.Unscoped().
		Where("note_id IN (SELECT id FROM notes WHERE user_id = ?)", userID).
		Delete(&domain.NoteItem{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Permanently delete Notes (Parent)
	if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&domain.Note{}).Error; err != nil {
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
