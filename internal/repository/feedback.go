package repository

import (
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

type FeedbackRepository interface {
	CreateFeedback(f domain.Feedback) (domain.Feedback, error)
}

type GormFeedbackRepository struct {
	Db *gorm.DB
}

func (r *GormFeedbackRepository) CreateFeedback(f domain.Feedback) (domain.Feedback, error) {
	err := r.Db.Create(&f).Error
	return f, err
}
