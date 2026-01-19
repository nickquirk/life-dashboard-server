package service

import (
	"context"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/repository"
	"gorm.io/gorm"
)

type Service interface {
	// User
	CreateUser(user domain.CreateUserRequest) (domain.CreateUserResponse, error)
	GetUser(domain.GetUserRequest) (domain.GetUserResponse, error)
	// Lists
	SyncTaskLists(ctx context.Context, userID uint) error
	GetTaskLists(userID uint) ([]domain.TaskList, error)
	// Tasks
	CreateTask(ctx context.Context, userID uint, taskListID string, req domain.CreateTaskRequest) (domain.Task, error)
	GetTasks(ctx context.Context, taskListID string) ([]domain.Task, error)
	SyncTasks(ctx context.Context, userID uint, taskListID string) error
	GetActiveTasks(ctx context.Context, taskListID string) ([]domain.Task, error)
	UpdateTask(ctx context.Context, userID uint, taskID string, req domain.UpdateTaskRequest) error
}

type service struct {
	userRepo repository.UserRepository
	taskRepo repository.TaskRepository
}

// service handler
func NewService(db *gorm.DB) Service {
	return &service{
		userRepo: repository.GormUserRepository{
			Db: db,
		},
		taskRepo: &repository.GormTaskRepository{
			Db: db,
		},
	}
}

func (s service) CreateUser(user domain.CreateUserRequest) (domain.CreateUserResponse, error) {
	resp, err := s.userRepo.Create(user)
	if err != nil {
		return domain.CreateUserResponse{}, err
	}
	return resp, nil
}

func (s service) GetUser(user domain.GetUserRequest) (domain.GetUserResponse, error) {
	resp, err := s.userRepo.Get(user)
	if err != nil {
		return domain.GetUserResponse{}, err
	}
	return resp, nil
}
