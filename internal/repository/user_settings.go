package repository

import (
	"errors"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserSettingsRepository interface {
	Get(userID uint) (domain.Settings, error)
	// Update loads the current settings, hands them to mutate, and persists the
	// result inside one transaction, so concurrent writes from two open tabs
	// can't clobber each other.
	Update(userID uint, mutate func(*domain.Settings) error) (domain.Settings, error)
}

type GormUserSettingsRepository struct {
	Db *gorm.DB
}

// Get returns defaults rather than ErrRecordNotFound for a user who has never
// saved anything, so callers don't each have to special-case the missing row.
func (r *GormUserSettingsRepository) Get(userID uint) (domain.Settings, error) {
	var row domain.UserSettings
	err := r.Db.Where("user_id = ?", userID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.DefaultSettings(), nil
	}
	if err != nil {
		return domain.Settings{}, err
	}
	return row.Data, nil
}

func (r *GormUserSettingsRepository) Update(userID uint, mutate func(*domain.Settings) error) (domain.Settings, error) {
	var out domain.Settings

	err := r.Db.Transaction(func(tx *gorm.DB) error {
		q := tx.Where("user_id = ?", userID)
		if tx.Dialector.Name() == "mysql" {
			// SELECT ... FOR UPDATE. SQLite (used by these tests) can't parse it
			// and doesn't need it — its transactions already serialise writes.
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}

		var row domain.UserSettings
		err := q.First(&row).Error
		existing := true
		if errors.Is(err, gorm.ErrRecordNotFound) {
			existing = false
			row = domain.UserSettings{UserID: userID, Data: domain.DefaultSettings()}
		} else if err != nil {
			return err
		}

		if err := mutate(&row.Data); err != nil {
			return err
		}
		out = row.Data

		if existing {
			return tx.Model(&row).Update("data", row.Data).Error
		}
		// OnConflict covers the narrow race where two requests both miss on the
		// first-ever write for a user.
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"data", "updated_at"}),
		}).Create(&row).Error
	})

	return out, err
}
