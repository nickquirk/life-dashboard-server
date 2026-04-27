package mocks

import (
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

// MockRoutineRepository implements repository.RoutineRepository with function fields.
type MockRoutineRepository struct {
	CreateRoutineFunc        func(domain.Routine) (domain.Routine, error)
	GetRoutinesByUserIDFunc  func(userID uint) ([]domain.RoutineWithStats, error)
	UpdateRoutineFunc        func(userID, routineID uint, updates map[string]interface{}) error
	DeleteRoutineFunc        func(userID, routineID uint) error
	CreateInstanceFunc       func(domain.RoutineInstance) (domain.RoutineInstance, error)
	GetInstancesByUserIDFunc func(userID uint, start, end time.Time) ([]domain.RoutineInstance, error)
	UpdateInstanceFunc       func(userID, instanceID uint, updates map[string]interface{}) error
	DeleteInstanceFunc       func(userID, instanceID uint) error
}

func (m *MockRoutineRepository) CreateRoutine(r domain.Routine) (domain.Routine, error) {
	if m.CreateRoutineFunc != nil {
		return m.CreateRoutineFunc(r)
	}
	return domain.Routine{}, nil
}

func (m *MockRoutineRepository) GetRoutinesByUserID(userID uint) ([]domain.RoutineWithStats, error) {
	if m.GetRoutinesByUserIDFunc != nil {
		return m.GetRoutinesByUserIDFunc(userID)
	}
	return nil, nil
}

func (m *MockRoutineRepository) UpdateRoutine(userID, routineID uint, updates map[string]interface{}) error {
	if m.UpdateRoutineFunc != nil {
		return m.UpdateRoutineFunc(userID, routineID, updates)
	}
	return nil
}

func (m *MockRoutineRepository) DeleteRoutine(userID, routineID uint) error {
	if m.DeleteRoutineFunc != nil {
		return m.DeleteRoutineFunc(userID, routineID)
	}
	return nil
}

func (m *MockRoutineRepository) CreateInstance(ri domain.RoutineInstance) (domain.RoutineInstance, error) {
	if m.CreateInstanceFunc != nil {
		return m.CreateInstanceFunc(ri)
	}
	return domain.RoutineInstance{}, nil
}

func (m *MockRoutineRepository) GetInstancesByUserID(userID uint, start, end time.Time) ([]domain.RoutineInstance, error) {
	if m.GetInstancesByUserIDFunc != nil {
		return m.GetInstancesByUserIDFunc(userID, start, end)
	}
	return nil, nil
}

func (m *MockRoutineRepository) UpdateInstance(userID, instanceID uint, updates map[string]interface{}) error {
	if m.UpdateInstanceFunc != nil {
		return m.UpdateInstanceFunc(userID, instanceID, updates)
	}
	return nil
}

func (m *MockRoutineRepository) DeleteInstance(userID, instanceID uint) error {
	if m.DeleteInstanceFunc != nil {
		return m.DeleteInstanceFunc(userID, instanceID)
	}
	return nil
}
