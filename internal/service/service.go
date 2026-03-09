package service

import (
	"context"

	"github.com/nickquirk/life-dashboard-server/internal/crypto"
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
	UpdateAppRefreshToken(userID uint, hashedToken string) error
	GetAppRefreshToken(userID uint) (string, error)
	DeleteAccount(userID uint) error
	// ------ Lists ---------
	SyncTaskLists(ctx context.Context, req domain.SyncTaskListsRequest) (domain.SyncTaskListsResponse, error)
	GetTaskLists(req domain.GetTaskListsRequest) (domain.GetTaskListsResponse, error)
	// ------ Tasks ---------
	CreateTask(ctx context.Context, req domain.CreateTaskRequest) (domain.CreateTaskResponse, error)
	GetTasks(ctx context.Context, req domain.GetTasksRequest) (domain.GetTasksResponse, error)
	SyncTasks(ctx context.Context, req domain.SyncTasksRequest) (domain.SyncTasksResponse, error)
	UpdateTask(ctx context.Context, req domain.UpdateTaskRequest) (domain.UpdateTaskResponse, error)
	DeleteTask(ctx context.Context, req domain.DeleteTaskRequest) (domain.DeleteTaskResponse, error)
	// ------ Calendar ---------
	GetCalendarEvents(ctx context.Context, req domain.GetCalendarEventsRequest) (domain.GetCalendarEventsResponse, error)
	// ------ Zones ---------
	CreateZone(req domain.CreateZoneRequest) (domain.CreateZoneResponse, error)
	GetZones(req domain.GetZonesRequest) (domain.GetZonesResponse, error)
	UpdateZone(req domain.UpdateZoneRequest) (domain.UpdateZoneResponse, error)
	DeleteZone(req domain.DeleteZoneRequest) (domain.DeleteZoneResponse, error)
	// ------ Routines ---------

	// ------ Feedback ---------
	CreateFeedback(req domain.CreateFeedbackRequest) (domain.CreateFeedbackResponse, error)
	// ------ Scratchpad ---------
	GetScratchpad(req domain.GetScratchpadRequest) (domain.GetScratchpadResponse, error)
	UpsertScratchpad(req domain.UpsertScratchpadRequest) (domain.UpsertScratchpadResponse, error)
}

type service struct {
	db             *gorm.DB
	userRepo       repository.UserRepository
	taskRepo       repository.TaskRepository
	calendarRepo   repository.CalendarRepository
	zoneRepo       repository.ZoneRepository
	routineRepo    repository.RoutineRepository
	feedbackRepo   repository.FeedbackRepository
	scratchpadRepo repository.ScratchpadRepository
}

// NewServiceWithRepos creates a Service with injected repositories
func NewServiceWithRepos(userRepo repository.UserRepository, taskRepo repository.TaskRepository,
	calendarRepo repository.CalendarRepository, zoneRepo repository.ZoneRepository, routineRepo repository.RoutineRepository, feedbackRepo repository.FeedbackRepository, scratchpadRepo repository.ScratchpadRepository) Service {
	return &service{userRepo: userRepo, taskRepo: taskRepo, calendarRepo: calendarRepo, zoneRepo: zoneRepo, routineRepo: routineRepo, feedbackRepo: feedbackRepo, scratchpadRepo: scratchpadRepo}
}

func (s service) Ping() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// service handler
func NewService(db *gorm.DB, encryptor crypto.TokenEncryptor) Service {
	return &service{
		db: db,
		userRepo: repository.GormUserRepository{
			Db:        db,
			Encryptor: encryptor,
		},
		taskRepo: &repository.GormTaskRepository{
			Db: db,
		},
		calendarRepo: &repository.GormCalendarRepository{
			Db: db,
		},
		zoneRepo:       &repository.GormZoneRepository{Db: db},
		routineRepo:    &repository.GormRoutineRepository{Db: db},
		feedbackRepo:   &repository.GormFeedbackRepository{Db: db},
		scratchpadRepo: &repository.GormScratchpadRepository{Db: db},
	}
}
