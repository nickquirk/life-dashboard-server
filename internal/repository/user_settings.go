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
		// Ensure a row exists before locking it (a no-op if one already does).
		// SELECT ... FOR UPDATE on a user_id that has no row yet takes an
		// InnoDB gap lock rather than a record lock, so two requests racing a
		// user's first-ever write can each hold a gap lock and deadlock on
		// insert. Upserting first means the locking SELECT below always finds
		// a real row to lock instead of an empty gap.
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoNothing: true,
		}).Create(&domain.UserSettings{UserID: userID, Data: domain.DefaultSettings()}).Error; err != nil {
			return err
		}

		q := tx.Where("user_id = ?", userID)
		if tx.Dialector.Name() == "mysql" {
			// SELECT ... FOR UPDATE. SQLite (used by these tests) can't parse it
			// and doesn't need it — its transactions already serialise writes.
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}

		var row domain.UserSettings
		if err := q.First(&row).Error; err != nil {
			return err
		}

		if err := mutate(&row.Data); err != nil {
			return err
		}
		out = row.Data

		return tx.Model(&row).Update("data", row.Data).Error
	})

	return out, err
}
