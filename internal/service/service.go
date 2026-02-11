package service

import (
	"context"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/repository"
	"gorm.io/gorm"
)

type Service interface {
	// ------ User ---------
	CreateUser(user domain.CreateUserRequest) (domain.CreateUserResponse, error)
	GetUser(domain.GetUserRequest) (domain.GetUserResponse, error)
	// ------ Lists ---------
	SyncTaskLists(ctx context.Context, userID uint) error
	GetTaskLists(userID uint) ([]domain.TaskList, error)
	// ------ Tasks ---------
	CreateTask(ctx context.Context, userID uint, taskListID string, req domain.CreateTaskRequest) (domain.Task, error)
	GetTasks(ctx context.Context, taskListID string) ([]domain.Task, error)
	SyncTasks(ctx context.Context, userID uint, taskListID string) error
	UpdateTask(ctx context.Context, userID uint, taskID string, req domain.UpdateTaskRequest) error
	DeleteTask(ctx context.Context, userID uint, taskID string) error
	// ------ Calendar ---------
	GetCalendarEvents(ctx context.Context, req domain.GetCalendarEventsRequest) (domain.GetCalendarEventsResponse, error)
}

type service struct {
	userRepo     repository.UserRepository
	taskRepo     repository.TaskRepository
	calendarRepo repository.CalendarRepository
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
		calendarRepo: &repository.GormCalendarRepository{
			Db: db,
		},
	}
}
