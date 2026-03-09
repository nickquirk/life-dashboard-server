package repository

import (
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

type RoutineRepository interface {
	CreateRoutine(domain.Routine) (domain.Routine, error)
	GetRoutinesByUserID(userID uint) ([]domain.Routine, error)
	UpdateRoutine(userID, routineID uint, updates map[string]interface{}) error
	DeleteRoutine(userID, routineID uint) error

	CreateInstance(domain.RoutineInstance) (domain.RoutineInstance, error)
	GetInstancesByUserID(userID uint, start, end time.Time) ([]domain.RoutineInstance, error)
	UpdateInstance(userID, instanceID uint, updates map[string]interface{}) error
	DeleteInstance(userID, instanceID uint) error
}

type GormRoutineRepository struct {
	Db *gorm.DB
}

// --- ROUTINES ---
func (r *GormRoutineRepository) CreateRoutine(routine domain.Routine) (domain.Routine, error) {
	err := r.Db.Create(&routine).Error
	return routine, err
}

func (r *GormRoutineRepository) GetRoutinesByUserID(userID uint) ([]domain.Routine, error) {
	var routines []domain.Routine
	err := r.Db.Where("user_id = ?", userID).Find(&routines).Error
	return routines, err
}

func (r *GormRoutineRepository) UpdateRoutine(userID, routineID uint, updates map[string]interface{}) error {
	result := r.Db.Model(&domain.Routine{}).Where("id = ? AND user_id = ?", routineID, userID).Updates(updates)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *GormRoutineRepository) DeleteRoutine(userID, routineID uint) error {
	result := r.Db.Where("id = ? AND user_id = ?", routineID, userID).Delete(&domain.Routine{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// --- INSTANCES ---
func (r *GormRoutineRepository) CreateInstance(instance domain.RoutineInstance) (domain.RoutineInstance, error) {
	err := r.Db.Create(&instance).Error
	return instance, err
}

func (r *GormRoutineRepository) GetInstancesByUserID(userID uint, start, end time.Time) ([]domain.RoutineInstance, error) {
	var instances []domain.RoutineInstance
	err := r.Db.Preload("Routine").
		Where("user_id = ? AND date >= ? AND date <= ?", userID, start, end).
		Find(&instances).Error
	return instances, err
}

func (r *GormRoutineRepository) UpdateInstance(userID, instanceID uint, updates map[string]interface{}) error {
	result := r.Db.Model(&domain.RoutineInstance{}).Where("id = ? AND user_id = ?", instanceID, userID).Updates(updates)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *GormRoutineRepository) DeleteInstance(userID, instanceID uint) error {
	result := r.Db.Where("id = ? AND user_id = ?", instanceID, userID).Delete(&domain.RoutineInstance{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}
