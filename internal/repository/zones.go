package repository

import (
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

type ZoneRepository interface {
	Create(zone domain.Zone) (domain.Zone, error)
}

type GormZoneRepository struct {
	Db *gorm.DB
}

func (r *GormZoneRepository) Create(zone domain.Zone) (domain.Zone, error) {
	err := r.Db.Create(&zone).Error
	return zone, err
}
