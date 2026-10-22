package mocks

import (
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

// MockProjectRepository implements repository.ProjectRepository with function fields.
// Nil fields return zero values.
type MockProjectRepository struct {
	CreateProjectFunc         func(p domain.Project) (domain.Project, error)
	GetProjectsByUserIDFunc   func(userID uint, archived bool) ([]domain.ProjectWithStats, error)
	GetProjectByIDFunc        func(userID, projectID uint) (domain.Project, error)
	UpdateProjectFunc         func(userID, projectID uint, updates map[string]interface{}) error
	DeleteProjectFunc         func(userID, projectID uint) error
	CreateActionFunc          func(a domain.Action) (domain.Action, error)
	GetActionByIDFunc         func(userID, actionID uint) (domain.Action, error)
	GetActionsByProjectIDFunc func(userID, projectID uint) ([]domain.Action, error)
	GetActionsInRangeFunc     func(userID uint, start, end time.Time) ([]domain.Action, error)
	GetReadyActionsFunc       func(userID uint) ([]domain.Action, error)
	UpdateActionFunc          func(userID, actionID uint, updates map[string]interface{}) error
	NextPositionFunc          func(userID, projectID uint, stage domain.Stage) (int, error)
	FirstPositionFunc         func(userID, projectID uint, stage domain.Stage) (int, error)
	ReorderActionsFunc        func(userID, projectID uint, stage domain.Stage, orderedIDs []uint) error
	ReplaceWithActionsFunc    func(userID, actionID uint, children []domain.Action) ([]domain.Action, error)
	DeleteActionFunc          func(userID, actionID uint) error
}

func (m *MockProjectRepository) CreateProject(p domain.Project) (domain.Project, error) {
	if m.CreateProjectFunc != nil {
		return m.CreateProjectFunc(p)
	}
	return domain.Project{}, nil
}

func (m *MockProjectRepository) GetProjectsByUserID(userID uint, archived bool) ([]domain.ProjectWithStats, error) {
	if m.GetProjectsByUserIDFunc != nil {
		return m.GetProjectsByUserIDFunc(userID, archived)
	}
	return nil, nil
}

func (m *MockProjectRepository) GetProjectByID(userID, projectID uint) (domain.Project, error) {
	if m.GetProjectByIDFunc != nil {
		return m.GetProjectByIDFunc(userID, projectID)
	}
	return domain.Project{}, nil
}

func (m *MockProjectRepository) UpdateProject(userID, projectID uint, updates map[string]interface{}) error {
	if m.UpdateProjectFunc != nil {
		return m.UpdateProjectFunc(userID, projectID, updates)
	}
	return nil
}

func (m *MockProjectRepository) DeleteProject(userID, projectID uint) error {
	if m.DeleteProjectFunc != nil {
		return m.DeleteProjectFunc(userID, projectID)
	}
	return nil
}

func (m *MockProjectRepository) CreateAction(a domain.Action) (domain.Action, error) {
	if m.CreateActionFunc != nil {
		return m.CreateActionFunc(a)
	}
	return domain.Action{}, nil
}

func (m *MockProjectRepository) GetActionByID(userID, actionID uint) (domain.Action, error) {
	if m.GetActionByIDFunc != nil {
		return m.GetActionByIDFunc(userID, actionID)
	}
	return domain.Action{}, nil
}

func (m *MockProjectRepository) GetActionsByProjectID(userID, projectID uint) ([]domain.Action, error) {
	if m.GetActionsByProjectIDFunc != nil {
		return m.GetActionsByProjectIDFunc(userID, projectID)
	}
	return nil, nil
}

func (m *MockProjectRepository) GetActionsInRange(userID uint, start, end time.Time) ([]domain.Action, error) {
	if m.GetActionsInRangeFunc != nil {
		return m.GetActionsInRangeFunc(userID, start, end)
	}
	return nil, nil
}

func (m *MockProjectRepository) GetReadyActions(userID uint) ([]domain.Action, error) {
	if m.GetReadyActionsFunc != nil {
		return m.GetReadyActionsFunc(userID)
	}
	return nil, nil
}

func (m *MockProjectRepository) UpdateAction(userID, actionID uint, updates map[string]interface{}) error {
	if m.UpdateActionFunc != nil {
		return m.UpdateActionFunc(userID, actionID, updates)
	}
	return nil
}

func (m *MockProjectRepository) NextPosition(userID, projectID uint, stage domain.Stage) (int, error) {
	if m.NextPositionFunc != nil {
		return m.NextPositionFunc(userID, projectID, stage)
	}
	return 0, nil
}

func (m *MockProjectRepository) FirstPosition(userID, projectID uint, stage domain.Stage) (int, error) {
	if m.FirstPositionFunc != nil {
		return m.FirstPositionFunc(userID, projectID, stage)
	}
	return 0, nil
}

func (m *MockProjectRepository) ReorderActions(userID, projectID uint, stage domain.Stage, orderedIDs []uint) error {
	if m.ReorderActionsFunc != nil {
		return m.ReorderActionsFunc(userID, projectID, stage, orderedIDs)
	}
	return nil
}

func (m *MockProjectRepository) ReplaceWithActions(userID, actionID uint, children []domain.Action) ([]domain.Action, error) {
	if m.ReplaceWithActionsFunc != nil {
		return m.ReplaceWithActionsFunc(userID, actionID, children)
	}
	return nil, nil
}

func (m *MockProjectRepository) DeleteAction(userID, actionID uint) error {
	if m.DeleteActionFunc != nil {
		return m.DeleteActionFunc(userID, actionID)
	}
	return nil
}
