package service

import (
	"context"
	"errors"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/crypto"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/repository"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type Service interface {
	// ------ Health ---------
	Ping() error
	// ------ User ---------
	CreateUser(user domain.CreateUserRequest) (domain.CreateUserResponse, error)
	GetUser(domain.GetUserRequest) (domain.GetUserResponse, error)
	GetUserEmail(userID uint) (string, error)
	CreateSession(userID uint, hashedToken string, expiresAt time.Time) error
	ValidateSession(userID uint, hashedToken string) (bool, error)
	DeleteSession(userID uint, hashedToken string) error
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
	DeleteTasks(ctx context.Context, req domain.DeleteTasksRequest) (domain.DeleteTasksResponse, error)
	// ------ Calendar ---------
	GetCalendarEvents(ctx context.Context, req domain.GetCalendarEventsRequest) (domain.GetCalendarEventsResponse, error)
	// ------ Zones ---------
	CreateZone(req domain.CreateZoneRequest) (domain.CreateZoneResponse, error)
	GetZones(req domain.GetZonesRequest) (domain.GetZonesResponse, error)
	UpdateZone(req domain.UpdateZoneRequest) (domain.UpdateZoneResponse, error)
	DeleteZone(req domain.DeleteZoneRequest) (domain.DeleteZoneResponse, error)
	// ------ Routines ---------
	CreateRoutine(req domain.CreateRoutineRequest) (domain.CreateRoutineResponse, error)
	GetRoutines(req domain.GetRoutineRequest) (domain.GetRoutineResponse, error)
	UpdateRoutine(req domain.UpdateRoutineRequest) (domain.UpdateRoutineResponse, error)
	DeleteRoutine(req domain.DeleteRoutineRequest) (domain.DeleteRoutineResponse, error)
	CreateRoutineInstance(req domain.CreateRoutineInstanceRequest) (domain.CreateRoutineInstanceResponse, error)
	GetRoutineInstances(req domain.GetRoutineInstancesRequest) (domain.GetRoutineInstancesResponse, error)
	UpdateRoutineInstance(req domain.UpdateRoutineInstanceRequest) (domain.UpdateRoutineInstanceResponse, error)
	DeleteRoutineInstance(req domain.DeleteRoutineInstanceRequest) (domain.DeleteRoutineInstanceResponse, error)
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

func (s *service) isTokenError(err error) bool {
	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) {
		return true
	}
	return false
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
