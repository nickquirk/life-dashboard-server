package service

import (
	"context"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/repository"
	"gorm.io/gorm"
)

type Service interface {
	// ------ Health ---------
	Ping() error
	// ------ User ---------
	CreateUser(user domain.CreateUserRequest) (domain.CreateUserResponse, error)
	GetUser(domain.GetUserRequest) (domain.GetUserResponse, error)
	SyncAllUsers(ctx context.Context) error
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
	// ------ Zones ---------
	CreateZone(req domain.CreateZoneRequest) (domain.CreateZoneResponse, error)
	GetZones(req domain.GetZonesRequest) (domain.GetZonesResponse, error)
	UpdateZone(req domain.UpdateZoneRequest) (domain.UpdateZoneResponse, error)
	DeleteZone(req domain.DeleteZonesRequest) (domain.DeleteZonesResponse, error)
}

type service struct {
	db           *gorm.DB
	userRepo     repository.UserRepository
	taskRepo     repository.TaskRepository
	calendarRepo repository.CalendarRepository
	zoneRepo     repository.ZoneRepository
}

func (s service) Ping() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// service handler
func NewService(db *gorm.DB) Service {
	return &service{
		db: db,
		userRepo: repository.GormUserRepository{
			Db: db,
		},
		taskRepo: &repository.GormTaskRepository{
			Db: db,
		},
		calendarRepo: &repository.GormCalendarRepository{
			Db: db,
		},
		zoneRepo: &repository.GormZoneRepository{Db: db},
	}
}
