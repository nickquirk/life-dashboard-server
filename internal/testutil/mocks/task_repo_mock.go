package mocks

import (
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/repository"
	"gorm.io/gorm"
)

// MockTaskRepository implements repository.TaskRepository with function fields.
type MockTaskRepository struct {
	UpsertTaskListsFunc             func([]domain.TaskList) error
	GetTaskListsFunc                func(userID uint) ([]domain.TaskList, error)
	GetTaskListFunc                 func(listID string) (domain.TaskList, error)
	UpdateListLastSyncFunc          func(listID string, t time.Time) error
	VerifyTaskListOwnerFunc         func(userID uint, taskListID string) error
	CreateTaskFunc                  func(domain.Task) error
	GetTasksFunc                    func(taskListID string) ([]domain.Task, error)
	GetTaskByIDFunc                 func(taskID string) (domain.Task, error)
	GetTaskListIDForTaskFunc        func(taskID string) (string, error)
	UpsertTasksFunc                 func([]domain.Task) error
	UpdateTaskFunc                  func(taskID string, updates map[string]interface{}) error
	DeleteTaskFunc                  func(taskID string) error
	DeleteTasksFunc                 func(taskIDs []string) error
	MarkTasksCompletedExcludingFunc func(taskListID string, activeIDs []string) error
	BeginTxFunc                     func() *gorm.DB
	WithTxFunc                      func(tx *gorm.DB) repository.TaskRepository
}

func (m *MockTaskRepository) UpsertTaskLists(lists []domain.TaskList) error {
	if m.UpsertTaskListsFunc != nil {
		return m.UpsertTaskListsFunc(lists)
	}
	return nil
}

func (m *MockTaskRepository) GetTaskLists(userID uint) ([]domain.TaskList, error) {
	if m.GetTaskListsFunc != nil {
		return m.GetTaskListsFunc(userID)
	}
	return nil, nil
}

func (m *MockTaskRepository) GetTaskList(listID string) (domain.TaskList, error) {
	if m.GetTaskListFunc != nil {
		return m.GetTaskListFunc(listID)
	}
	return domain.TaskList{}, nil
}

func (m *MockTaskRepository) UpdateListLastSync(listID string, t time.Time) error {
	if m.UpdateListLastSyncFunc != nil {
		return m.UpdateListLastSyncFunc(listID, t)
	}
	return nil
}

func (m *MockTaskRepository) VerifyTaskListOwner(userID uint, taskListID string) error {
	if m.VerifyTaskListOwnerFunc != nil {
		return m.VerifyTaskListOwnerFunc(userID, taskListID)
	}
	return nil
}

func (m *MockTaskRepository) CreateTask(task domain.Task) error {
	if m.CreateTaskFunc != nil {
		return m.CreateTaskFunc(task)
	}
	return nil
}

func (m *MockTaskRepository) GetTasks(taskListID string) ([]domain.Task, error) {
	if m.GetTasksFunc != nil {
		return m.GetTasksFunc(taskListID)
	}
	return nil, nil
}

func (m *MockTaskRepository) GetTaskByID(taskID string) (domain.Task, error) {
	if m.GetTaskByIDFunc != nil {
		return m.GetTaskByIDFunc(taskID)
	}
	return domain.Task{}, nil
}

func (m *MockTaskRepository) GetTaskListIDForTask(taskID string) (string, error) {
	if m.GetTaskListIDForTaskFunc != nil {
		return m.GetTaskListIDForTaskFunc(taskID)
	}
	return "", nil
}

func (m *MockTaskRepository) UpsertTasks(tasks []domain.Task) error {
	if m.UpsertTasksFunc != nil {
		return m.UpsertTasksFunc(tasks)
	}
	return nil
}

func (m *MockTaskRepository) UpdateTask(taskID string, updates map[string]interface{}) error {
	if m.UpdateTaskFunc != nil {
		return m.UpdateTaskFunc(taskID, updates)
	}
	return nil
}

func (m *MockTaskRepository) DeleteTask(taskID string) error {
	if m.DeleteTaskFunc != nil {
		return m.DeleteTaskFunc(taskID)
	}
	return nil
}

func (m *MockTaskRepository) DeleteTasks(taskIDs []string) error {
	if m.DeleteTasksFunc != nil {
		return m.DeleteTasksFunc(taskIDs)
	}
	return nil
}

func (m *MockTaskRepository) MarkTasksCompletedExcluding(taskListID string, activeIDs []string) error {
	if m.MarkTasksCompletedExcludingFunc != nil {
		return m.MarkTasksCompletedExcludingFunc(taskListID, activeIDs)
	}
	return nil
}

func (m *MockTaskRepository) BeginTx() *gorm.DB {
	if m.BeginTxFunc != nil {
		return m.BeginTxFunc()
	}
	return nil
}

func (m *MockTaskRepository) WithTx(tx *gorm.DB) repository.TaskRepository {
	if m.WithTxFunc != nil {
		return m.WithTxFunc(tx)
	}
	return m
}
