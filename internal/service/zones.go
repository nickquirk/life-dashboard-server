package service

import (
	"encoding/json"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func (s *service) CreateZone(req domain.CreateZoneRequest) (domain.CreateZoneResponse, error) {
	newZone := domain.Zone{
		UserID:     req.UserID,
		Label:      req.Label,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Color:      req.Color,
		DaysActive: req.DaysActive,
	}

	created, err := s.zoneRepo.Create(newZone)
	if err != nil {
		return domain.CreateZoneResponse{}, err
	}

	return domain.CreateZoneResponse{
		ID: created.ID,
	}, nil
}

func (s *service) GetZones(req domain.GetZonesRequest) (domain.GetZonesResponse, error) {
	zones, err := s.zoneRepo.GetByUserID(req.UserID)
	if err != nil {
		return domain.GetZonesResponse{}, err
	}

	response := domain.GetZonesResponse{Zones: []domain.ZoneResponse{}}
	for _, z := range zones {
		response.Zones = append(response.Zones, domain.ZoneResponse{
			ID:         z.ID,
			Label:      z.Label,
			StartTime:  z.StartTime,
			EndTime:    z.EndTime,
			Color:      z.Color,
			DaysActive: z.DaysActive,
		})
	}
	return response, nil
}

func (s *service) UpdateZone(req domain.UpdateZoneRequest) (domain.UpdateZoneResponse, error) {
	updates := make(map[string]interface{})

	// Use DB keys
	if req.Label != nil {
		updates["label"] = *req.Label
	}
	if req.StartTime != nil {
		updates["start_time"] = *req.StartTime
	}
	if req.EndTime != nil {
		updates["end_time"] = *req.EndTime
	}
	if req.Color != nil {
		updates["color"] = *req.Color
	}
	// manually serialise so that its in the right form for the db
	if req.DaysActive != nil {
		jsonData, err := json.Marshal(req.DaysActive)
		if err != nil {
			return domain.UpdateZoneResponse{}, err
		}
		updates["days_active"] = string(jsonData)
	}

	if len(updates) == 0 {
		return domain.UpdateZoneResponse{}, nil
	}

	zone, err := s.zoneRepo.Update(req.UserID, req.ID, updates)
	if err != nil {
		return domain.UpdateZoneResponse{}, err
	}

	return domain.UpdateZoneResponse{
		ID:         zone.ID,
		UserID:     zone.UserID,
		Label:      zone.Label,
		StartTime:  zone.StartTime,
		EndTime:    zone.EndTime,
		Color:      zone.Color,
		DaysActive: zone.DaysActive,
		UpdatedAt:  zone.UpdatedAt,
	}, nil

}

func (s *service) DeleteZone(req domain.DeleteZoneRequest) (domain.DeleteZoneResponse, error) {
	err := s.zoneRepo.Delete(req.UserID, req.ID)
	if err != nil {
		return domain.DeleteZoneResponse{}, err
	}

	return domain.DeleteZoneResponse{
		ID: req.ID,
	}, nil
}
