package repository

import (
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ScratchpadRepository interface {
	Get(userID uint, date string) (domain.Scratchpad, error)
	Upsert(scratchpad domain.Scratchpad) error
}

type GormScratchpadRepository struct {
	Db *gorm.DB
}

func (r *GormScratchpadRepository) Get(userID uint, date string) (domain.Scratchpad, error) {
	var note domain.Scratchpad
	err := r.Db.Where("user_id = ? AND date = ?", userID, date).First(&note).Error
	return note, err
}

func (r *GormScratchpadRepository) Upsert(note domain.Scratchpad) error {
	return r.Db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"content", "updated_at"}),
	}).Create(&note).Error
}
