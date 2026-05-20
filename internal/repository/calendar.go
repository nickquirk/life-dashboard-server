package repository

import (
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

type GormCalendarRepository struct {
	Db *gorm.DB
}

type CalendarRepository interface {
	GetEvents(userID uint, start, end time.Time) ([]domain.CalendarEvent, error)
	// Transaction support
	BeginTx() *gorm.DB
	WithTx(tx *gorm.DB) CalendarRepository
}

func (r *GormCalendarRepository) GetEvents(userID uint, start, end time.Time) ([]domain.CalendarEvent, error) {
	events := make([]domain.CalendarEvent, 0)
	// Fetch events overlapping the requested window
	err := r.Db.Where("user_id = ? AND start >= ? AND end <= ?", userID, start, end).
		Order("start asc").
		Find(&events).Error
	return events, err
}

// Start a transaction
func (r *GormCalendarRepository) BeginTx() *gorm.DB {
	return r.Db.Begin()
}

// Return a copy of the repository using the transaction
func (r *GormCalendarRepository) WithTx(tx *gorm.DB) CalendarRepository {
	return &GormCalendarRepository{
		Db: tx, // The new instance uses the transaction connection
	}
}
