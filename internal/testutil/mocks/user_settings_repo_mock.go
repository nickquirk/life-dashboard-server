package mocks

import "github.com/nickquirk/life-dashboard-server/internal/domain"

// MockUserSettingsRepository implements repository.UserSettingsRepository with
// function fields.
type MockUserSettingsRepository struct {
	GetFunc    func(userID uint) (domain.Settings, error)
	UpdateFunc func(userID uint, mutate func(*domain.Settings) error) (domain.Settings, error)
}

func (m *MockUserSettingsRepository) Get(userID uint) (domain.Settings, error) {
	if m.GetFunc != nil {
		return m.GetFunc(userID)
	}
	return domain.DefaultSettings(), nil
}

// Update's default behaviour runs the mutation against a fresh default document,
// mirroring a real first write. Service tests then exercise the real merge and
// validation instead of stubbing past them.
func (m *MockUserSettingsRepository) Update(userID uint, mutate func(*domain.Settings) error) (domain.Settings, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(userID, mutate)
	}
	cur := domain.DefaultSettings()
	if err := mutate(&cur); err != nil {
		return domain.Settings{}, err
	}
	return cur, nil
}
