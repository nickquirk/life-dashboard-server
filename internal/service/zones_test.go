package service

import (
	"errors"
	"testing"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newZoneService(zoneRepo *mocks.MockZoneRepository) Service {
	return NewServiceWithRepos(nil, nil, nil, zoneRepo, nil, nil)
}

// --- CreateZone ---

func TestService_CreateZone_Success(t *testing.T) {
	repo := &mocks.MockZoneRepository{
		CreateFunc: func(z domain.Zone) (domain.Zone, error) {
			z.ID = 10
			return z, nil
		},
	}
	svc := newZoneService(repo)

	resp, err := svc.CreateZone(domain.CreateZoneRequest{
		UserID: 1, Label: "Work", StartTime: "09:00", EndTime: "17:00", Color: "blue",
	})
	require.NoError(t, err)
	assert.Equal(t, uint(10), resp.ID)
}

func TestService_CreateZone_Error(t *testing.T) {
	repo := &mocks.MockZoneRepository{
		CreateFunc: func(z domain.Zone) (domain.Zone, error) {
			return domain.Zone{}, errors.New("db error")
		},
	}
	svc := newZoneService(repo)

	_, err := svc.CreateZone(domain.CreateZoneRequest{UserID: 1, Label: "Work"})
	assert.Error(t, err)
}

// --- GetZones ---

func TestService_GetZones_Success(t *testing.T) {
	repo := &mocks.MockZoneRepository{
		GetByUserIDFunc: func(userID uint) ([]domain.Zone, error) {
			return []domain.Zone{
				{Label: "Work", StartTime: "09:00", EndTime: "17:00", Color: "blue", DaysActive: []uint{1, 2, 3}},
				{Label: "Gym", StartTime: "18:00", EndTime: "19:00", Color: "green", DaysActive: []uint{1, 3, 5}},
			}, nil
		},
	}
	svc := newZoneService(repo)

	resp, err := svc.GetZones(domain.GetZonesRequest{UserID: 1})
	require.NoError(t, err)
	assert.Len(t, resp.Zones, 2)
	assert.Equal(t, "Work", resp.Zones[0].Label)
	assert.Equal(t, []uint{1, 2, 3}, resp.Zones[0].DaysActive)
}

func TestService_GetZones_Empty(t *testing.T) {
	repo := &mocks.MockZoneRepository{
		GetByUserIDFunc: func(userID uint) ([]domain.Zone, error) {
			return nil, nil
		},
	}
	svc := newZoneService(repo)

	resp, err := svc.GetZones(domain.GetZonesRequest{UserID: 1})
	require.NoError(t, err)
	assert.Nil(t, resp.Zones)
}

func TestService_GetZones_Error(t *testing.T) {
	repo := &mocks.MockZoneRepository{
		GetByUserIDFunc: func(userID uint) ([]domain.Zone, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newZoneService(repo)

	_, err := svc.GetZones(domain.GetZonesRequest{UserID: 1})
	assert.Error(t, err)
}

// --- UpdateZone ---

func TestService_UpdateZone_AllFields(t *testing.T) {
	label := "Updated"
	startTime := "10:00"
	endTime := "18:00"
	color := "red"

	var capturedUpdates map[string]interface{}
	repo := &mocks.MockZoneRepository{
		UpdateFunc: func(userID uint, zoneID uint, updates map[string]interface{}) (domain.Zone, error) {
			capturedUpdates = updates
			return domain.Zone{Label: label, StartTime: startTime, EndTime: endTime, Color: color}, nil
		},
	}
	svc := newZoneService(repo)

	resp, err := svc.UpdateZone(domain.UpdateZoneRequest{
		ID: 1, UserID: 1,
		Label: &label, StartTime: &startTime, EndTime: &endTime, Color: &color,
		DaysActive: []uint{1, 2},
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated", resp.Label)
	assert.Contains(t, capturedUpdates, "label")
	assert.Contains(t, capturedUpdates, "start_time")
	assert.Contains(t, capturedUpdates, "end_time")
	assert.Contains(t, capturedUpdates, "color")
	assert.Contains(t, capturedUpdates, "days_active")
}

func TestService_UpdateZone_PartialFields(t *testing.T) {
	label := "Just Label"
	repo := &mocks.MockZoneRepository{
		UpdateFunc: func(userID uint, zoneID uint, updates map[string]interface{}) (domain.Zone, error) {
			assert.Len(t, updates, 1)
			assert.Equal(t, "Just Label", updates["label"])
			return domain.Zone{Label: label}, nil
		},
	}
	svc := newZoneService(repo)

	_, err := svc.UpdateZone(domain.UpdateZoneRequest{ID: 1, UserID: 1, Label: &label})
	assert.NoError(t, err)
}

func TestService_UpdateZone_NoFields(t *testing.T) {
	svc := newZoneService(&mocks.MockZoneRepository{})

	resp, err := svc.UpdateZone(domain.UpdateZoneRequest{ID: 1, UserID: 1})
	require.NoError(t, err)
	assert.Equal(t, domain.UpdateZoneResponse{}, resp)
}

func TestService_UpdateZone_Error(t *testing.T) {
	label := "Fail"
	repo := &mocks.MockZoneRepository{
		UpdateFunc: func(userID uint, zoneID uint, updates map[string]interface{}) (domain.Zone, error) {
			return domain.Zone{}, errors.New("db error")
		},
	}
	svc := newZoneService(repo)

	_, err := svc.UpdateZone(domain.UpdateZoneRequest{ID: 1, UserID: 1, Label: &label})
	assert.Error(t, err)
}

// --- DeleteZone ---

func TestService_DeleteZone_Success(t *testing.T) {
	repo := &mocks.MockZoneRepository{
		DeleteFunc: func(userID uint, zoneID uint) error {
			assert.Equal(t, uint(1), userID)
			assert.Equal(t, uint(5), zoneID)
			return nil
		},
	}
	svc := newZoneService(repo)

	resp, err := svc.DeleteZone(domain.DeleteZoneRequest{ID: 5, UserID: 1})
	require.NoError(t, err)
	assert.Equal(t, uint(5), resp.ID)
}

func TestService_DeleteZone_Error(t *testing.T) {
	repo := &mocks.MockZoneRepository{
		DeleteFunc: func(userID uint, zoneID uint) error {
			return errors.New("not found")
		},
	}
	svc := newZoneService(repo)

	_, err := svc.DeleteZone(domain.DeleteZoneRequest{ID: 999, UserID: 1})
	assert.Error(t, err)
}
