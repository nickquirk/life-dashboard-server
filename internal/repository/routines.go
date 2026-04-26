package repository

import (
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

type RoutineRepository interface {
	CreateRoutine(domain.Routine) (domain.Routine, error)
	GetRoutinesByUserID(userID uint) ([]domain.RoutineWithStats, error)
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

func (r *GormRoutineRepository) GetRoutinesByUserID(userID uint) ([]domain.RoutineWithStats, error) {
	var routines []domain.RoutineWithStats

	now := time.Now()
	weekStart := startOfISOWeek(now)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Stats are scoped to the routine's reset period:
	//   reset_period IS NULL  -> lifetime
	//   'weekly'              -> instances since Monday of current week
	//   'monthly'             -> instances since the 1st of current month
	inPeriod := `(
		r.reset_period IS NULL
		OR (r.reset_period = 'weekly'  AND ri.date >= ?)
		OR (r.reset_period = 'monthly' AND ri.date >= ?)
	)`

	err := r.Db.
		Table("routines AS r").
		Select(`
			r.*,
			COALESCE(SUM(CASE WHEN ri.id IS NOT NULL AND `+inPeriod+`
			THEN COALESCE(ri.duration_mins, r.duration_mins)
			ELSE 0 END), 0) AS scheduled_mins,
			COALESCE(SUM(CASE WHEN ri.id IS NOT NULL AND ri.status = 'completed' AND `+inPeriod+`
				THEN COALESCE(ri.duration_mins, r.duration_mins)
				ELSE 0 END), 0) AS completed_mins,
			COALESCE(SUM(CASE WHEN ri.id IS NOT NULL AND `+inPeriod+`
				THEN 1 ELSE 0 END), 0) AS instance_count,
			COALESCE(SUM(CASE WHEN ri.id IS NOT NULL AND ri.status = 'completed' AND `+inPeriod+`
				THEN 1 ELSE 0 END), 0) AS completed_count
		`,
			weekStart, monthStart,
			weekStart, monthStart,
			weekStart, monthStart,
			weekStart, monthStart,
		).
		Joins("LEFT JOIN routine_instances AS ri ON ri.routine_id = r.id AND ri.deleted_at IS NULL").
		Where("r.user_id = ? AND r.deleted_at IS NULL", userID).
		Group("r.id").
		Scan(&routines).Error
	return routines, err
}

// startOfISOWeek returns midnight on the Monday of t's week, in t's location.
func startOfISOWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday counts as the last day of the prior ISO week
	}
	monday := t.AddDate(0, 0, -(weekday - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
}

func (r *GormRoutineRepository) UpdateRoutine(userID, routineID uint, updates map[string]interface{}) error {
	result := r.Db.Model(&domain.Routine{}).Where("id = ? AND user_id = ?", routineID, userID).Updates(updates)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

func (r *GormRoutineRepository) DeleteRoutine(userID, routineID uint) error {
	result := r.Db.Unscoped().Where("id = ? AND user_id = ?", routineID, userID).Delete(&domain.Routine{})
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
	result := r.Db.Unscoped().Where("id = ? AND user_id = ?", instanceID, userID).Delete(&domain.RoutineInstance{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}
