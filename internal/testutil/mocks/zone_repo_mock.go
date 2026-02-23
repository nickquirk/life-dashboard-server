package mocks

import "github.com/nickquirk/life-dashboard-server/internal/domain"

// MockZoneRepository implements repository.ZoneRepository with function fields.
type MockZoneRepository struct {
	CreateFunc      func(domain.Zone) (domain.Zone, error)
	GetByUserIDFunc func(userID uint) ([]domain.Zone, error)
	UpdateFunc      func(userID uint, zoneID uint, updates map[string]interface{}) (domain.Zone, error)
	DeleteFunc      func(userID uint, zoneID uint) error
}

func (m *MockZoneRepository) Create(zone domain.Zone) (domain.Zone, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(zone)
	}
	return domain.Zone{}, nil
}

func (m *MockZoneRepository) GetByUserID(userID uint) ([]domain.Zone, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(userID)
	}
	return nil, nil
}

func (m *MockZoneRepository) Update(userID uint, zoneID uint, updates map[string]interface{}) (domain.Zone, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(userID, zoneID, updates)
	}
	return domain.Zone{}, nil
}

func (m *MockZoneRepository) Delete(userID uint, zoneID uint) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(userID, zoneID)
	}
	return nil
}
