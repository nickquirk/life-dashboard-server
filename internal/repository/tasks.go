package repository

import (
	"fmt"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormTaskRepository struct {
	Db *gorm.DB
}

type TaskRepository interface {
	// Task Lists
	UpsertTaskLists(lists []domain.TaskList) error
	GetTaskLists(userID uint) ([]domain.TaskList, error)
	GetTaskList(listID string) (domain.TaskList, error)
	UpdateListLastSync(listID string, t time.Time) error
	VerifyTaskListOwner(userID uint, taskListID string) error
	// Tasks
	CreateTask(task domain.Task) error
	GetTasks(taskListID string) ([]domain.Task, error)
	GetTaskByID(taskID string) (domain.Task, error)
	GetTaskListIDForTask(taskID string) (string, error)
	UpsertTasks(tasks []domain.Task) error
	UpdateTask(taskID string, updates map[string]interface{}) error
	DeleteTask(taskID string) error
	DeleteTasks(taskIDs []string) error
	MarkTasksCompletedExcluding(taskListID string, activeIDs []string) error
	ReorderSubtasks(userID uint, parentTaskID string, orderedIDs []string) error
	// Transaction support
	BeginTx() *gorm.DB
	WithTx(tx *gorm.DB) TaskRepository
}

func (r *GormTaskRepository) UpsertTaskLists(lists []domain.TaskList) error {
	if len(lists) == 0 {
		return nil
	}
	// Try to create these records. If a record with this ID already exists, update it with the new data instead of failing
	return r.Db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}}, // Google ID is primary
		DoUpdates: clause.AssignmentColumns([]string{"title", "updated"}),
	}).Create(&lists).Error
}

func (r *GormTaskRepository) GetTaskLists(userID uint) ([]domain.TaskList, error) {
	lists := make([]domain.TaskList, 0)
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

func (r *GormTaskRepository) VerifyTaskListOwner(userID uint, taskListID string) error {
	var id string
	// only fetch the ID column
	err := r.Db.Model(&domain.TaskList{}).
		Select("id").
		Where("id = ? AND user_id = ?", taskListID, userID).
		First(&id).Error

	return err
}

func (r *GormTaskRepository) UpsertTasks(tasks []domain.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	return r.Db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "status", "due", "notes", "updated", "task_list_id"}),
	}).Create(&tasks).Error
}

func (r *GormTaskRepository) CreateTask(task domain.Task) error {
	return r.Db.Create(&task).Error
}

func (r *GormTaskRepository) GetTasks(taskListID string) ([]domain.Task, error) {
	tasks := make([]domain.Task, 0)
	err := r.Db.
		Where("task_list_id = ?", taskListID).
		Order("position ASC, created_at ASC").
		Find(&tasks).Error
	return tasks, err
}

func (r *GormTaskRepository) GetTaskByID(taskID string) (domain.Task, error) {
	var task domain.Task
	err := r.Db.First(&task, "id = ?", taskID).Error
	return task, err
}

func (r *GormTaskRepository) GetTaskListIDForTask(taskID string) (string, error) {
	var taskListID string
	err := r.Db.Model(&domain.Task{}).
		Select("task_list_id").
		Where("id = ?", taskID).
		First(&taskListID).Error
	return taskListID, err
}

func (r *GormTaskRepository) UpdateTask(taskID string, updates map[string]interface{}) error {
	return r.Db.Model(&domain.Task{}).Where("id = ?", taskID).Updates(updates).Error
}

func (r *GormTaskRepository) DeleteTask(taskID string) error {
	return r.Db.Unscoped().Where("id = ?", taskID).Delete(&domain.Task{}).Error
}

func (r *GormTaskRepository) DeleteTasks(taskIDs []string) error {
	if len(taskIDs) == 0 {
		return nil
	}
	return r.Db.Unscoped().Where("id IN ?", taskIDs).Delete(&domain.Task{}).Error
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

func (r *GormTaskRepository) ReorderSubtasks(userID uint, parentTaskID string, orderedIDs []string) error {
	return r.Db.Transaction(func(tx *gorm.DB) error {
		// Verify the parent exists and belongs to a task list owned by this user.
		var parent domain.Task
		if err := tx.First(&parent, "id = ?", parentTaskID).Error; err != nil {
			return gorm.ErrRecordNotFound
		}
		var ownedListID string
		if err := tx.Model(&domain.TaskList{}).
			Select("id").
			Where("id = ? AND user_id = ?", parent.TaskListID, userID).
			First(&ownedListID).Error; err != nil {
			return gorm.ErrRecordNotFound
		}

		if len(orderedIDs) > 0 {
			var count int64
			if err := tx.Model(&domain.Task{}).
				Where("id IN ? AND parent = ?", orderedIDs, parentTaskID).
				Count(&count).Error; err != nil {
				return err
			}
			if count != int64(len(orderedIDs)) {
				return fmt.Errorf("%w: some task IDs are not subtasks of this parent", domain.ErrInvalidInput)
			}
		}
		for i, id := range orderedIDs {
			if err := tx.Model(&domain.Task{}).Where("id = ?", id).Update("position", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Start a transaction
func (r *GormTaskRepository) BeginTx() *gorm.DB {
	return r.Db.Begin()
}

// Return a copy of the repository using the transaction
func (r *GormTaskRepository) WithTx(tx *gorm.DB) TaskRepository {
	return &GormTaskRepository{
		Db: tx, // This new repo instance uses the transaction connection
	}
}
