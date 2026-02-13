package service

import (
	"context"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func (s *service) CreateZone(ctx context.Context, req domain.CreateZoneRequest) (domain.CreateZoneResponse, error) {
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
		ID:        created.ID,
		Label:     created.Label,
		StartTime: created.StartTime,
		EndTime:   created.EndTime,
		Color:     created.Color,
	}, nil
}
