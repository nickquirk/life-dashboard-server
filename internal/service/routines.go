package service

import "github.com/nickquirk/life-dashboard-server/internal/domain"

func (s *service) CreateRoutine(req domain.CreateRoutineRequest) (domain.CreateRoutineResponse, error) {
	created, err := s.routineRepo.CreateRoutine(domain.Routine{
		UserID:          req.UserID,
		Title:           req.Title,
		DurationMins:    req.DurationMins,
		TargetTotalMins: req.TargetTotalMins,
		ResetPeriod:     req.ResetPeriod,
	})
	if err != nil {
		return domain.CreateRoutineResponse{}, err
	}
	return domain.CreateRoutineResponse{
		ID:              created.ID,
		Title:           created.Title,
		DurationMins:    created.DurationMins,
		TargetTotalMins: created.TargetTotalMins,
		ResetPeriod:     created.ResetPeriod,
	}, nil
}

func (s *service) GetRoutines(req domain.GetRoutineRequest) (domain.GetRoutineResponse, error) {
	routines, err := s.routineRepo.GetRoutinesByUserID(req.UserID)
	if err != nil {
		return domain.GetRoutineResponse{}, err
	}
	return domain.GetRoutineResponse{
		Routines: routines,
	}, nil
}

func (s *service) UpdateRoutine(req domain.UpdateRoutineRequest) (domain.UpdateRoutineResponse, error) {
	updates := make(map[string]interface{})

	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.DurationMins != nil {
		updates["duration_mins"] = *req.DurationMins
	}
	// If TTM is set then routine becomes a time goal
	if req.TargetTotalMins != nil {
		// If FE sends zero then convert back to regular routine
		if *req.TargetTotalMins == 0 {
			updates["target_total_mins"] = nil
		} else {
			updates["target_total_mins"] = *req.TargetTotalMins
		}
	}
	if req.ResetPeriod != nil {
		if *req.ResetPeriod == "" || *req.ResetPeriod == "one_off" {
			updates["reset_period"] = nil
		} else {
			updates["reset_period"] = *req.ResetPeriod
		}
	}

	if len(updates) == 0 {
		return domain.UpdateRoutineResponse{}, nil
	}

	err := s.routineRepo.UpdateRoutine(req.UserID, req.ID, updates)
	if err != nil {
		return domain.UpdateRoutineResponse{}, err
	}

	return domain.UpdateRoutineResponse{
		ID: req.ID,
	}, nil
}

func (s *service) DeleteRoutine(req domain.DeleteRoutineRequest) (domain.DeleteRoutineResponse, error) {
	err := s.routineRepo.DeleteRoutine(req.UserID, req.ID)
	if err != nil {
		return domain.DeleteRoutineResponse{}, err
	}
	return domain.DeleteRoutineResponse{
		ID: req.ID,
	}, nil
}

func (s *service) CreateRoutineInstance(req domain.CreateRoutineInstanceRequest) (domain.CreateRoutineInstanceResponse, error) {
	created, err := s.routineRepo.CreateInstance(domain.RoutineInstance{
		UserID:       req.UserID,
		RoutineID:    req.RoutineID,
		Date:         req.Date,
		Status:       "needsAction",
		DurationMins: req.DurationMins,
	})
	if err != nil {
		return domain.CreateRoutineInstanceResponse{}, err
	}
	return domain.CreateRoutineInstanceResponse{
		ID:           created.ID,
		RoutineID:    created.RoutineID,
		Routine:      created.Routine,
		Date:         created.Date,
		Status:       created.Status,
		DurationMins: created.DurationMins,
	}, nil
}

func (s *service) GetRoutineInstances(req domain.GetRoutineInstancesRequest) (domain.GetRoutineInstancesResponse, error) {
	instances, err := s.routineRepo.GetInstancesByUserID(req.UserID, req.Start, req.End)
	if err != nil {
		return domain.GetRoutineInstancesResponse{}, err
	}
	return domain.GetRoutineInstancesResponse{
		Instances: instances,
	}, nil
}

func (s *service) UpdateRoutineInstance(req domain.UpdateRoutineInstanceRequest) (domain.UpdateRoutineInstanceResponse, error) {
	updates := make(map[string]interface{})

	if req.Date != nil {
		updates["date"] = *req.Date
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.DurationMins != nil {
		updates["duration_mins"] = *req.DurationMins
	}

	if len(updates) == 0 {
		return domain.UpdateRoutineInstanceResponse{}, nil
	}

	err := s.routineRepo.UpdateInstance(req.UserID, req.ID, updates)
	if err != nil {
		return domain.UpdateRoutineInstanceResponse{}, err
	}

	return domain.UpdateRoutineInstanceResponse{
		ID: req.ID,
	}, nil
}

func (s *service) DeleteRoutineInstance(req domain.DeleteRoutineInstanceRequest) (domain.DeleteRoutineInstanceResponse, error) {
	err := s.routineRepo.DeleteInstance(req.UserID, req.ID)
	if err != nil {
		return domain.DeleteRoutineInstanceResponse{}, err
	}
	return domain.DeleteRoutineInstanceResponse{
		ID: req.ID,
	}, nil
}
