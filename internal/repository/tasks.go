package repository

import (
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormTaskRepository struct {
	Db *gorm.DB
}

type TaskRepository interface {
	UpsertTaskLists(lists []domain.TaskList) error
	GetTaskLists(userID uint) ([]domain.TaskList, error)
	GetTaskList(listID string) (domain.TaskList, error)
	UpdateListLastSync(listID string, t time.Time) error

	UpsertTasks(tasks []domain.Task) error
	GetTasks(taskListID string) ([]domain.Task, error)
	GetActiveTasks(taskListID string) ([]domain.Task, error)
	DeleteTasks(ids []string) error
	MarkTasksCompletedExcluding(taskListID string, activeIDs []string) error
}

func (r *GormTaskRepository) UpsertTaskLists(lists []domain.TaskList) error {
	if len(lists) == 0 {
		return nil
	}
	// Try to create these records. If a record with this ID already exists, update it with the new data instead of failing
	return r.Db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}}, // Google ID is primary
		DoUpdates: clause.AssignmentColumns([]string{"title", "updated", "last_sync"}),
	}).Create(&lists).Error
}

func (r *GormTaskRepository) GetTaskLists(userID uint) ([]domain.TaskList, error) {
	var lists []domain.TaskList
	err := r.Db.Where("user_id = ?", userID).Order("updated desc").Find(&lists).Error
	return lists, err
}

func (r *GormTaskRepository) GetTaskList(listID string) (domain.TaskList, error) {
	var list domain.TaskList
	err := r.Db.First(&list, "id = ?", listID).Error
	return list, err
}

func (r *GormTaskRepository) UpdateListLastSync(listID string, t time.Time) error {
	// Find task list and update last sync
	return r.Db.Model(&domain.TaskList{}).Where("id = ?", listID).Update("last_sync", t).Error
}

func (r *GormTaskRepository) UpsertTasks(tasks []domain.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	// Update title, status, due, notes, updated when conflict occurs
	// For now, let's assume we overwrite standard fields but we need to be careful not to wipe Quadrant
	// if we aren't setting it in the incoming struct.
	return r.Db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "status", "due", "notes", "updated", "task_list_id"}),
	}).Create(&tasks).Error
}

func (r *GormTaskRepository) GetTasks(taskListID string) ([]domain.Task, error) {
	var tasks []domain.Task
	err := r.Db.Where("task_list_id = ?", taskListID).Find(&tasks).Error
	return tasks, err
}

func (r *GormTaskRepository) DeleteTasks(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.Db.Where("id IN ?", ids).Delete(&domain.Task{}).Error
}

func (r *GormTaskRepository) MarkTasksCompletedExcluding(taskListID string, activeIDs []string) error {
	// 1. If Google returns NO active tasks, mark ALL tasks in this list as completed
	if len(activeIDs) == 0 {
		return r.Db.Model(&domain.Task{}).
			Where("task_list_id = ? AND status != ?", taskListID, "completed").
			Update("status", "completed").Error
	}

	// 2. Otherwise, find tasks in this list that are NOT in the activeIDs group
	//    and update their status to 'completed'.
	return r.Db.Model(&domain.Task{}).
		Where("task_list_id = ?", taskListID).
		Not("id IN ?", activeIDs).
		Where("status != ?", "completed"). // Optimization: don't update if already completed
		Update("status", "completed").Error
}

func (r *GormTaskRepository) GetActiveTasks(taskListID string) ([]domain.Task, error) {
	var tasks []domain.Task
	// filter by task_list_id AND status
	err := r.Db.Where("task_list_id = ? AND status = ?", taskListID, "needsAction").
		Find(&tasks).Error
	return tasks, err
}
