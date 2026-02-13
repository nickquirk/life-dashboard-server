package repository

import (
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

type ZoneRepository interface {
	Create(zone domain.CreateZoneRequest) (domain.Zone, error)
}

type GormZoneRepository struct {
	Db *gorm.DB
}

func (r *GormZoneRepository) Create(req domain.CreateZoneRequest) (domain.Zone, error) {
	zone := domain.Zone{
		UserID:    req.UserID,
		Label:     req.Label,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Color:     req.Color,
	}
	err := r.Db.Create(&zone).Error
	if err != nil {
		return domain.Zone{}, err
	}
	return zone, err
}
