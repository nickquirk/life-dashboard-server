package service

import "github.com/nickquirk/life-dashboard-server/internal/domain"

func (s *service) CreateRoutine(req domain.CreateRoutineRequest) (domain.CreateRoutineResponse, error) {
	created, err := s.routineRepo.CreateRoutine(domain.Routine{
		UserID:       req.UserID,
		Title:        req.Title,
		DurationMins: req.DurationMins,
	})
	if err != nil {
		return domain.CreateRoutineResponse{}, err
	}
	return domain.CreateRoutineResponse{
		ID:           created.ID,
		Title:        created.Title,
		DurationMins: created.DurationMins,
	}, nil
}
