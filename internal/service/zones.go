package service

import (
	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func (s *service) CreateZone(req domain.CreateZoneRequest) (domain.CreateZoneResponse, error) {
	newZone := domain.Zone{
		UserID:    req.UserID,
		Label:     req.Label,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Color:     req.Color,
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

	var response domain.GetZonesResponse
	for _, z := range zones {
		response.Zones = append(response.Zones, domain.Zone{
			Model:     z.Model, // This includes ID, CreatedAt, etc.
			Label:     z.Label,
			StartTime: z.StartTime,
			EndTime:   z.EndTime,
			Color:     z.Color,
		})
	}
	return response, nil
}

func (s *service) DeleteZone(req domain.DeleteZonesRequest) (domain.DeleteZonesResponse, error) {
	err := s.zoneRepo.Delete(req.UserID, req.ID)
	if err != nil {
		return domain.DeleteZonesResponse{}, err
	}

	return domain.DeleteZonesResponse{
		Message: "Zone deleted successfully",
	}, nil
}
