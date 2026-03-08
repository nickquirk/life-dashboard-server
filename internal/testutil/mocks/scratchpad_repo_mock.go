package mocks

import "github.com/nickquirk/life-dashboard-server/internal/domain"

// MockScratchpadRepository implements repository.ScratchpadRepository with function fields.
type MockScratchpadRepository struct {
	GetFunc    func(userID uint, date string) (domain.Scratchpad, error)
	UpsertFunc func(scratchpad domain.Scratchpad) error
}

func (m *MockScratchpadRepository) Get(userID uint, date string) (domain.Scratchpad, error) {
	if m.GetFunc != nil {
		return m.GetFunc(userID, date)
	}
	return domain.Scratchpad{}, nil
}

func (m *MockScratchpadRepository) Upsert(scratchpad domain.Scratchpad) error {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(scratchpad)
	}
	return nil
}
