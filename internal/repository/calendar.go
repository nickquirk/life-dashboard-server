package repository

import (
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

type GormCalendarRepository struct {
	Db *gorm.DB
}

type CalendarRepository interface {
	GetCalendarEvents(userID uint) ([]domain.CalendarEvent, error)
}

func (r *GormCalendarRepository) GetCalendarEvents(userID uint) ([]domain.CalendarEvent, error) {
	var events []domain.CalendarEvent

	err := r.Db.Where("user_id = ?", userID).Find(&events).Error
	return events, err
}
