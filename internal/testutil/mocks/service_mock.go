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
	GetUserEmailFunc        func(userID uint) (string, error)
	UpdateAppRefreshTokenFunc func(userID uint, hashedToken string) error
	GetAppRefreshTokenFunc  func(userID uint) (string, error)
	TriggerGlobalSyncFunc   func(ctx context.Context, projectID, location, queue, workerURL, serviceAccountEmail string) error
	SyncSingleUserFunc      func(ctx context.Context, userID uint) error
	SyncTaskListsFunc       func(ctx context.Context, req domain.SyncTaskListsRequest) (domain.SyncTaskListsResponse, error)
	GetTaskListsFunc        func(req domain.GetTaskListsRequest) (domain.GetTaskListsResponse, error)
	CreateTaskFunc          func(ctx context.Context, req domain.CreateTaskRequest) (domain.CreateTaskResponse, error)
	GetTasksFunc            func(ctx context.Context, req domain.GetTasksRequest) (domain.GetTasksResponse, error)
	SyncTasksFunc           func(ctx context.Context, req domain.SyncTasksRequest) (domain.SyncTasksResponse, error)
	UpdateTaskFunc          func(ctx context.Context, req domain.UpdateTaskRequest) (domain.UpdateTaskResponse, error)
	DeleteTaskFunc          func(ctx context.Context, req domain.DeleteTaskRequest) (domain.DeleteTaskResponse, error)
	GetCalendarEventsFunc   func(ctx context.Context, req domain.GetCalendarEventsRequest) (domain.GetCalendarEventsResponse, error)
	CreateZoneFunc          func(req domain.CreateZoneRequest) (domain.CreateZoneResponse, error)
	GetZonesFunc            func(req domain.GetZonesRequest) (domain.GetZonesResponse, error)
	UpdateZoneFunc          func(req domain.UpdateZoneRequest) (domain.UpdateZoneResponse, error)
	DeleteZoneFunc          func(req domain.DeleteZoneRequest) (domain.DeleteZoneResponse, error)
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

func (m *MockService) GetUserEmail(userID uint) (string, error) {
	if m.GetUserEmailFunc != nil {
		return m.GetUserEmailFunc(userID)
	}
	return "", nil
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

func (m *MockService) SyncTaskLists(ctx context.Context, req domain.SyncTaskListsRequest) (domain.SyncTaskListsResponse, error) {
	if m.SyncTaskListsFunc != nil {
		return m.SyncTaskListsFunc(ctx, req)
	}
	return domain.SyncTaskListsResponse{}, nil
}

func (m *MockService) GetTaskLists(req domain.GetTaskListsRequest) (domain.GetTaskListsResponse, error) {
	if m.GetTaskListsFunc != nil {
		return m.GetTaskListsFunc(req)
	}
	return domain.GetTaskListsResponse{}, nil
}

func (m *MockService) CreateTask(ctx context.Context, req domain.CreateTaskRequest) (domain.CreateTaskResponse, error) {
	if m.CreateTaskFunc != nil {
		return m.CreateTaskFunc(ctx, req)
	}
	return domain.CreateTaskResponse{}, nil
}

func (m *MockService) GetTasks(ctx context.Context, req domain.GetTasksRequest) (domain.GetTasksResponse, error) {
	if m.GetTasksFunc != nil {
		return m.GetTasksFunc(ctx, req)
	}
	return domain.GetTasksResponse{}, nil
}

func (m *MockService) SyncTasks(ctx context.Context, req domain.SyncTasksRequest) (domain.SyncTasksResponse, error) {
	if m.SyncTasksFunc != nil {
		return m.SyncTasksFunc(ctx, req)
	}
	return domain.SyncTasksResponse{}, nil
}

func (m *MockService) UpdateTask(ctx context.Context, req domain.UpdateTaskRequest) (domain.UpdateTaskResponse, error) {
	if m.UpdateTaskFunc != nil {
		return m.UpdateTaskFunc(ctx, req)
	}
	return domain.UpdateTaskResponse{}, nil
}

func (m *MockService) DeleteTask(ctx context.Context, req domain.DeleteTaskRequest) (domain.DeleteTaskResponse, error) {
	if m.DeleteTaskFunc != nil {
		return m.DeleteTaskFunc(ctx, req)
	}
	return domain.DeleteTaskResponse{}, nil
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

func (m *MockService) DeleteZone(req domain.DeleteZoneRequest) (domain.DeleteZoneResponse, error) {
	if m.DeleteZoneFunc != nil {
		return m.DeleteZoneFunc(req)
	}
	return domain.DeleteZoneResponse{}, nil
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

