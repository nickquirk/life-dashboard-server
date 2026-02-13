package repository

import (
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

type ZoneRepository interface {
	Create(domain.Zone) (domain.Zone, error)
	GetByUserID(userID uint) ([]domain.Zone, error)
}

type GormZoneRepository struct {
	Db *gorm.DB
}

func (r *GormZoneRepository) Create(zone domain.Zone) (domain.Zone, error) {
	err := r.Db.Create(&zone).Error
	return zone, err
}

func (r *GormZoneRepository) GetByUserID(userID uint) ([]domain.Zone, error) {
	var zones []domain.Zone
	err := r.Db.Where("user_id = ?", userID).Find(&zones).Error
	return zones, err
}
