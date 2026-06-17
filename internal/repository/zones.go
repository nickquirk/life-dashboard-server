package repository

import (
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ZoneRepository interface {
	Create(domain.Zone) (domain.Zone, error)
	GetByUserID(userID uint) ([]domain.Zone, error)
	Update(userID uint, zoneID uint, updates map[string]interface{}) (domain.Zone, error)
	Delete(userID uint, zoneID uint) error
}

type GormZoneRepository struct {
	Db *gorm.DB
}

func (r *GormZoneRepository) Create(zone domain.Zone) (domain.Zone, error) {
	err := r.Db.Create(&zone).Error
	return zone, err
}

func (r *GormZoneRepository) GetByUserID(userID uint) ([]domain.Zone, error) {
	zones := make([]domain.Zone, 0)
	err := r.Db.Where("user_id = ?", userID).Find(&zones).Error
	return zones, err
}

func (r *GormZoneRepository) Update(userID uint, zoneID uint, updates map[string]interface{}) (domain.Zone, error) {
	// Ensure the zone belongs to the user before updating
	var zone domain.Zone
	result := r.Db.Model(&zone).
		Clauses(clause.Returning{}).
		Where("id = ? AND user_id = ?", zoneID, userID).
		Updates(updates)

	if result.Error != nil {
		return domain.Zone{}, result.Error
	}
	if result.RowsAffected == 0 {
		return domain.Zone{}, gorm.ErrRecordNotFound
	}
	return zone, nil
}

func (r *GormZoneRepository) Delete(userID uint, zoneID uint) error {
	result := r.Db.Unscoped().Where("id = ? AND user_id = ?", zoneID, userID).Delete(&domain.Zone{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
