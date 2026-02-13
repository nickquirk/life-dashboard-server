package service

import (
	"context"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func (s *service) CreateZone(ctx context.Context, req domain.CreateZoneRequest) (domain.CreateZoneResponse, error) {
	zone, err := s.zoneRepo.Create(req)
	if err != nil {
		return domain.CreateZoneResponse{}, err
	}

	return domain.CreateZoneResponse{
		ID: zone.ID,
	}, nil
}
