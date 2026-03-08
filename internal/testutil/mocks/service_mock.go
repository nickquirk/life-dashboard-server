package mocks

import (
	"context"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

// MockService implements service.Service with function fields.
// Nil fields return zero values.
type MockService struct {
	PingFunc                func() error
	CreateUserFunc          func(domain.CreateUserRequest) (domain.CreateUserResponse, error)
	GetUserFunc             func(domain.GetUserRequest) (domain.GetUserResponse, error)
	UpdateAppRefreshTokenFunc func(userID uint, hashedToken string) error
	GetAppRefreshTokenFunc  func(userID uint) (string, error)
	TriggerGlobalSyncFunc   func(ctx context.Context, projectID, location, queue, workerURL, serviceAccountEmail string) error
	SyncSingleUserFunc      func(ctx context.Context, userID uint) error
	SyncTaskListsFunc       func(ctx context.Context, userID uint) error
	GetTaskListsFunc        func(userID uint) ([]domain.TaskList, error)
	CreateTaskFunc          func(ctx context.Context, userID uint, taskListID string, req domain.CreateTaskRequest) (domain.Task, error)
	GetTasksFunc            func(ctx context.Context, userID uint, taskListID string) ([]domain.Task, error)
	SyncTasksFunc           func(ctx context.Context, userID uint, taskListID string) error
	UpdateTaskFunc          func(ctx context.Context, userID uint, taskID string, req domain.UpdateTaskRequest) error
	DeleteTaskFunc          func(ctx context.Context, userID uint, taskID string) error
	GetCalendarEventsFunc   func(ctx context.Context, req domain.GetCalendarEventsRequest) (domain.GetCalendarEventsResponse, error)
	CreateZoneFunc          func(req domain.CreateZoneRequest) (domain.CreateZoneResponse, error)
	GetZonesFunc            func(req domain.GetZonesRequest) (domain.GetZonesResponse, error)
	UpdateZoneFunc          func(req domain.UpdateZoneRequest) (domain.UpdateZoneResponse, error)
	DeleteZoneFunc          func(req domain.DeleteZonesRequest) (domain.DeleteZonesResponse, error)
	DeleteAccountFunc       func(userID uint) error
	CreateFeedbackFunc func(req domain.CreateFeedbackRequest) (domain.CreateFeedbackResponse, error)
	GetScratchpadFunc    func(req domain.GetScratchpadRequest) (domain.GetScratchpadResponse, error)
	UpsertScratchpadFunc func(req domain.UpsertScratchpadRequest) (domain.UpsertScratchpadResponse, error)
}

func (m *MockService) Ping() error {
	if m.PingFunc != nil {
		return m.PingFunc()
	}
	return nil
}

func (m *MockService) CreateUser(req domain.CreateUserRequest) (domain.CreateUserResponse, error) {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(req)
	}
	return domain.CreateUserResponse{}, nil
}

func (m *MockService) GetUser(req domain.GetUserRequest) (domain.GetUserResponse, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(req)
	}
	return domain.GetUserResponse{}, nil
}

func (m *MockService) UpdateAppRefreshToken(userID uint, hashedToken string) error {
	if m.UpdateAppRefreshTokenFunc != nil {
		return m.UpdateAppRefreshTokenFunc(userID, hashedToken)
	}
	return nil
}

func (m *MockService) GetAppRefreshToken(userID uint) (string, error) {
	if m.GetAppRefreshTokenFunc != nil {
		return m.GetAppRefreshTokenFunc(userID)
	}
	return "", nil
}

func (m *MockService) TriggerGlobalSync(ctx context.Context, projectID, location, queue, workerURL, serviceAccountEmail string) error {
	if m.TriggerGlobalSyncFunc != nil {
		return m.TriggerGlobalSyncFunc(ctx, projectID, location, queue, workerURL, serviceAccountEmail)
	}
	return nil
}

func (m *MockService) SyncSingleUser(ctx context.Context, userID uint) error {
	if m.SyncSingleUserFunc != nil {
		return m.SyncSingleUserFunc(ctx, userID)
	}
	return nil
}

func (m *MockService) SyncTaskLists(ctx context.Context, userID uint) error {
	if m.SyncTaskListsFunc != nil {
		return m.SyncTaskListsFunc(ctx, userID)
	}
	return nil
}

func (m *MockService) GetTaskLists(userID uint) ([]domain.TaskList, error) {
	if m.GetTaskListsFunc != nil {
		return m.GetTaskListsFunc(userID)
	}
	return nil, nil
}

func (m *MockService) CreateTask(ctx context.Context, userID uint, taskListID string, req domain.CreateTaskRequest) (domain.Task, error) {
	if m.CreateTaskFunc != nil {
		return m.CreateTaskFunc(ctx, userID, taskListID, req)
	}
	return domain.Task{}, nil
}

func (m *MockService) GetTasks(ctx context.Context, userID uint, taskListID string) ([]domain.Task, error) {
	if m.GetTasksFunc != nil {
		return m.GetTasksFunc(ctx, userID, taskListID)
	}
	return nil, nil
}

func (m *MockService) SyncTasks(ctx context.Context, userID uint, taskListID string) error {
	if m.SyncTasksFunc != nil {
		return m.SyncTasksFunc(ctx, userID, taskListID)
	}
	return nil
}

func (m *MockService) UpdateTask(ctx context.Context, userID uint, taskID string, req domain.UpdateTaskRequest) error {
	if m.UpdateTaskFunc != nil {
		return m.UpdateTaskFunc(ctx, userID, taskID, req)
	}
	return nil
}

func (m *MockService) DeleteTask(ctx context.Context, userID uint, taskID string) error {
	if m.DeleteTaskFunc != nil {
		return m.DeleteTaskFunc(ctx, userID, taskID)
	}
	return nil
}

func (m *MockService) GetCalendarEvents(ctx context.Context, req domain.GetCalendarEventsRequest) (domain.GetCalendarEventsResponse, error) {
	if m.GetCalendarEventsFunc != nil {
		return m.GetCalendarEventsFunc(ctx, req)
	}
	return domain.GetCalendarEventsResponse{}, nil
}

func (m *MockService) CreateZone(req domain.CreateZoneRequest) (domain.CreateZoneResponse, error) {
	if m.CreateZoneFunc != nil {
		return m.CreateZoneFunc(req)
	}
	return domain.CreateZoneResponse{}, nil
}

func (m *MockService) GetZones(req domain.GetZonesRequest) (domain.GetZonesResponse, error) {
	if m.GetZonesFunc != nil {
		return m.GetZonesFunc(req)
	}
	return domain.GetZonesResponse{}, nil
}

func (m *MockService) UpdateZone(req domain.UpdateZoneRequest) (domain.UpdateZoneResponse, error) {
	if m.UpdateZoneFunc != nil {
		return m.UpdateZoneFunc(req)
	}
	return domain.UpdateZoneResponse{}, nil
}

func (m *MockService) DeleteZone(req domain.DeleteZonesRequest) (domain.DeleteZonesResponse, error) {
	if m.DeleteZoneFunc != nil {
		return m.DeleteZoneFunc(req)
	}
	return domain.DeleteZonesResponse{}, nil
}

func (m *MockService) DeleteAccount(userID uint) error {
	if m.DeleteAccountFunc != nil {
		return m.DeleteAccountFunc(userID)
	}
	return nil
}

func (m *MockService) CreateFeedback(req domain.CreateFeedbackRequest) (domain.CreateFeedbackResponse, error) {
	if m.CreateFeedbackFunc != nil {
		return m.CreateFeedbackFunc(req)
	}
	return domain.CreateFeedbackResponse{}, nil
}

func (m *MockService) GetScratchpad(req domain.GetScratchpadRequest) (domain.GetScratchpadResponse, error) {
	if m.GetScratchpadFunc != nil {
		return m.GetScratchpadFunc(req)
	}
	return domain.GetScratchpadResponse{}, nil
}

func (m *MockService) UpsertScratchpad(req domain.UpsertScratchpadRequest) (domain.UpsertScratchpadResponse, error) {
	if m.UpsertScratchpadFunc != nil {
		return m.UpsertScratchpadFunc(req)
	}
	return domain.UpsertScratchpadResponse{}, nil
}

