package service

import "github.com/nickquirk/life-dashboard-server/internal/domain"

func (s *service) CreateRoutine(req domain.CreateRoutineRequest) (domain.CreateRoutineResponse, error) {
	if err := req.Validate(); err != nil {
		return domain.CreateRoutineResponse{}, err
	}

	routine := domain.Routine{
		UserID:       req.UserID,
		Title:        req.Title,
		DurationMins: req.DurationMins,
		ResetPeriod:  req.ResetPeriod,
	}
	// Only persist a goal if Target > 0 (Target == 0 is the "no goal" / "clear" sentinel).
	if req.Goal != nil && req.Goal.Target > 0 {
		gTypeStr := string(req.Goal.Type) // *string is what GORM expects on the column
		gTarget := req.Goal.Target
		routine.GoalType = &gTypeStr
		routine.GoalTarget = &gTarget
		routine.Goal = &domain.Goal{Type: req.Goal.Type, Target: gTarget}
	}

	created, err := s.routineRepo.CreateRoutine(routine)
	if err != nil {
		return domain.CreateRoutineResponse{}, err
	}
	return domain.CreateRoutineResponse{
		ID:           created.ID,
		Title:        created.Title,
		DurationMins: created.DurationMins,
		Goal:         created.Goal,
		ResetPeriod:  created.ResetPeriod,
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
	if err := req.Validate(); err != nil {
		return domain.UpdateRoutineResponse{}, err
	}

	updates := make(map[string]interface{})

	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.DurationMins != nil {
		updates["duration_mins"] = *req.DurationMins
	}

	if req.IsArchived != nil {
		updates["is_archived"] = *req.IsArchived
	}

	// Track whether the caller is demoting this routine back to a regular one
	// so we can also strip the now-meaningless reset period.
	clearingTarget := false
	if req.TargetTotalMins != nil {
		if *req.TargetTotalMins == 0 {
			updates["target_total_mins"] = nil
			clearingTarget = true
		} else {
			updates["target_total_mins"] = *req.TargetTotalMins
		}
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
		if *req.ResetPeriod == "" || *req.ResetPeriod == domain.ResetPeriodOneOff {
			updates["reset_period"] = nil
		} else {
			updates["reset_period"] = *req.ResetPeriod
		}
	} else if clearingTarget {
		// The caller cleared the target without specifying a new reset period.
		// Don't leave 'weekly'/'monthly' stranded on a non-goal routine — it
		// would surprise the user if they later re-add a target.
		updates["reset_period"] = nil
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
