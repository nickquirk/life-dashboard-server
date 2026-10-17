package service

import (
	"fmt"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

// --- Stage transitions ---
//
// Every write that changes an Action's stage goes through one verb, and each
// verb is only allowed from the stages listed here. Together with the
// per-verb effects below, this table is the lifecycle:
//
//	ChangeStage  braindump ⇄ refine ⇄ ready   entering ready needs a duration
//	Schedule     ready → scheduled           sets the date
//	Move         scheduled → scheduled       changes the date
//	Unschedule   scheduled → ready           clears the date
//	Complete     any but done → done         keeps the date if it has one
//	Reopen       done → scheduled | ready | refine
//	BreakDown    braindump/refine/ready → several new actions in refine
//
// Invariant: date is set ⇔ stage is scheduled, or stage is done and the
// action was completed while scheduled.

type actionVerb string

const (
	verbChangeStage actionVerb = "change the stage of"
	verbSchedule    actionVerb = "schedule"
	verbMove        actionVerb = "move"
	verbUnschedule  actionVerb = "unschedule"
	verbComplete    actionVerb = "complete"
	verbReopen      actionVerb = "reopen"
	verbBreakDown   actionVerb = "break down"
)

var verbAllowedFrom = map[actionVerb][]domain.Stage{
	verbChangeStage: {domain.StageBraindump, domain.StageRefine, domain.StageReady},
	verbSchedule:    {domain.StageReady},
	verbMove:        {domain.StageScheduled},
	verbUnschedule:  {domain.StageScheduled},
	verbComplete:    {domain.StageBraindump, domain.StageRefine, domain.StageReady, domain.StageScheduled},
	verbReopen:      {domain.StageDone},
	verbBreakDown:   {domain.StageBraindump, domain.StageRefine, domain.StageReady},
}

func checkVerb(a domain.Action, v actionVerb) error {
	for _, s := range verbAllowedFrom[v] {
		if a.Stage == s {
			return nil
		}
	}
	return fmt.Errorf("%w: cannot %s an action in %s", domain.ErrInvalidInput, v, a.Stage)
}

// needsDuration reports whether an action in stage s must have a duration.
func needsDuration(s domain.Stage) bool {
	return s == domain.StageReady || s == domain.StageScheduled
}

// reopenStage is where a done action goes when it's unticked.
func reopenStage(a domain.Action) domain.Stage {
	switch {
	case a.Date != nil:
		return domain.StageScheduled
	case a.DurationMins != nil:
		return domain.StageReady
	default:
		return domain.StageRefine
	}
}

// --- Projects ---

func (s *service) CreateProject(req domain.CreateProjectRequest) (domain.Project, error) {
	if err := req.Validate(); err != nil {
		return domain.Project{}, err
	}
	return s.projectRepo.CreateProject(domain.Project{
		UserID: req.UserID,
		Title:  req.Title,
		Color:  req.Color,
	})
}

func (s *service) GetProjects(req domain.GetProjectsRequest) (domain.GetProjectsResponse, error) {
	projects, err := s.projectRepo.GetProjectsByUserID(req.UserID, req.Archived)
	if err != nil {
		return domain.GetProjectsResponse{}, err
	}
	return domain.GetProjectsResponse{Projects: projects}, nil
}

func (s *service) UpdateProject(req domain.UpdateProjectRequest) (domain.Project, error) {
	if err := req.Validate(); err != nil {
		return domain.Project{}, err
	}

	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Color != nil {
		updates["color"] = *req.Color
	}
	if req.IsArchived != nil {
		updates["is_archived"] = *req.IsArchived
	}

	if len(updates) > 0 {
		if err := s.projectRepo.UpdateProject(req.UserID, req.ID, updates); err != nil {
			return domain.Project{}, err
		}
	}
	return s.projectRepo.GetProjectByID(req.UserID, req.ID)
}

func (s *service) DeleteProject(req domain.DeleteProjectRequest) (domain.DeleteProjectResponse, error) {
	if err := s.projectRepo.DeleteProject(req.UserID, req.ID); err != nil {
		return domain.DeleteProjectResponse{}, err
	}
	return domain.DeleteProjectResponse{ID: req.ID}, nil
}

// --- Actions: reads ---

func (s *service) GetActions(req domain.GetActionsRequest) (domain.GetActionsResponse, error) {
	actions, err := s.projectRepo.GetActionsByProjectID(req.UserID, req.ProjectID)
	if err != nil {
		return domain.GetActionsResponse{}, err
	}
	return domain.GetActionsResponse{Actions: actions}, nil
}

func (s *service) GetScheduledActions(req domain.GetScheduledActionsRequest) (domain.GetActionsWithProjectResponse, error) {
	actions, err := s.projectRepo.GetActionsInRange(req.UserID, req.Start, req.End)
	if err != nil {
		return domain.GetActionsWithProjectResponse{}, err
	}
	return domain.GetActionsWithProjectResponse{Actions: withProjects(actions)}, nil
}

func (s *service) GetReadyActions(req domain.GetReadyActionsRequest) (domain.GetActionsWithProjectResponse, error) {
	actions, err := s.projectRepo.GetReadyActions(req.UserID)
	if err != nil {
		return domain.GetActionsWithProjectResponse{}, err
	}
	return domain.GetActionsWithProjectResponse{Actions: withProjects(actions)}, nil
}

func withProjects(actions []domain.Action) []domain.ActionWithProject {
	out := make([]domain.ActionWithProject, len(actions))
	for i, a := range actions {
		out[i] = domain.ActionWithProject{
			Action:  a,
			Project: domain.ProjectRef{ID: a.Project.ID, Title: a.Project.Title, Color: a.Project.Color},
		}
	}
	return out
}

// --- Actions: writes ---

func (s *service) CreateAction(req domain.CreateActionRequest) (domain.Action, error) {
	if err := req.Validate(); err != nil {
		return domain.Action{}, err
	}

	project, err := s.projectRepo.GetProjectByID(req.UserID, req.ProjectID)
	if err != nil {
		return domain.Action{}, err
	}
	if project.IsArchived {
		return domain.Action{}, fmt.Errorf("%w: cannot add actions to an archived project", domain.ErrInvalidInput)
	}

	position, err := s.projectRepo.NextPosition(req.UserID, req.ProjectID, domain.StageBraindump)
	if err != nil {
		return domain.Action{}, err
	}

	return s.projectRepo.CreateAction(domain.Action{
		UserID:       req.UserID,
		ProjectID:    req.ProjectID,
		Title:        req.Title,
		Notes:        req.Notes,
		DurationMins: req.DurationMins,
		Stage:        domain.StageBraindump,
		Position:     position,
	})
}

func (s *service) UpdateAction(req domain.UpdateActionRequest) (domain.Action, error) {
	if err := req.Validate(); err != nil {
		return domain.Action{}, err
	}

	current, err := s.projectRepo.GetActionByID(req.UserID, req.ID)
	if err != nil {
		return domain.Action{}, err
	}

	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Notes != nil {
		updates["notes"] = *req.Notes
	}
	if req.DurationMins.Set {
		if req.DurationMins.Value == nil && needsDuration(current.Stage) {
			return domain.Action{}, fmt.Errorf("%w: an action in %s must keep its duration", domain.ErrInvalidInput, current.Stage)
		}
		updates["duration_mins"] = req.DurationMins.Value
	}

	if len(updates) == 0 {
		return current, nil
	}
	return s.applyActionUpdates(req.UserID, req.ID, updates)
}

func (s *service) ChangeActionStage(req domain.ChangeStageRequest) (domain.Action, error) {
	if err := req.Validate(); err != nil {
		return domain.Action{}, err
	}

	current, err := s.projectRepo.GetActionByID(req.UserID, req.ID)
	if err != nil {
		return domain.Action{}, err
	}
	if err := checkVerb(current, verbChangeStage); err != nil {
		return domain.Action{}, err
	}
	if current.Stage == req.Stage {
		return domain.Action{}, fmt.Errorf("%w: action is already in %s", domain.ErrInvalidInput, req.Stage)
	}
	if needsDuration(req.Stage) && current.DurationMins == nil {
		return domain.Action{}, fmt.Errorf("%w: an action needs a duration before it can be %s", domain.ErrInvalidInput, req.Stage)
	}

	return s.moveToColumn(current, req.Stage, nil)
}

func (s *service) ScheduleAction(req domain.ScheduleActionRequest) (domain.Action, error) {
	if err := req.Validate(); err != nil {
		return domain.Action{}, err
	}

	current, err := s.projectRepo.GetActionByID(req.UserID, req.ID)
	if err != nil {
		return domain.Action{}, err
	}
	if err := checkVerb(current, verbSchedule); err != nil {
		return domain.Action{}, err
	}

	return s.applyActionUpdates(req.UserID, req.ID, map[string]interface{}{
		"stage": domain.StageScheduled,
		"date":  req.Date.UTC(),
	})
}

func (s *service) MoveAction(req domain.MoveActionRequest) (domain.Action, error) {
	if err := req.Validate(); err != nil {
		return domain.Action{}, err
	}

	current, err := s.projectRepo.GetActionByID(req.UserID, req.ID)
	if err != nil {
		return domain.Action{}, err
	}
	if err := checkVerb(current, verbMove); err != nil {
		return domain.Action{}, err
	}

	return s.applyActionUpdates(req.UserID, req.ID, map[string]interface{}{
		"date": req.Date.UTC(),
	})
}

func (s *service) UnscheduleAction(req domain.UnscheduleActionRequest) (domain.Action, error) {
	current, err := s.projectRepo.GetActionByID(req.UserID, req.ID)
	if err != nil {
		return domain.Action{}, err
	}
	if err := checkVerb(current, verbUnschedule); err != nil {
		return domain.Action{}, err
	}

	return s.moveToColumn(current, domain.StageReady, map[string]interface{}{"date": nil})
}

func (s *service) CompleteAction(req domain.CompleteActionRequest) (domain.Action, error) {
	current, err := s.projectRepo.GetActionByID(req.UserID, req.ID)
	if err != nil {
		return domain.Action{}, err
	}
	if err := checkVerb(current, verbComplete); err != nil {
		return domain.Action{}, err
	}

	// The date is left alone: a scheduled action stays on its calendar day,
	// drawn completed; an undated one stays undated.
	return s.applyActionUpdates(req.UserID, req.ID, map[string]interface{}{
		"stage": domain.StageDone,
	})
}

func (s *service) ReopenAction(req domain.ReopenActionRequest) (domain.Action, error) {
	current, err := s.projectRepo.GetActionByID(req.UserID, req.ID)
	if err != nil {
		return domain.Action{}, err
	}
	if err := checkVerb(current, verbReopen); err != nil {
		return domain.Action{}, err
	}

	to := reopenStage(current)
	if to == domain.StageScheduled {
		return s.applyActionUpdates(req.UserID, req.ID, map[string]interface{}{"stage": to})
	}
	return s.moveToColumn(current, to, nil)
}

func (s *service) BreakDownAction(req domain.BreakDownActionRequest) (domain.GetActionsResponse, error) {
	if err := req.Validate(); err != nil {
		return domain.GetActionsResponse{}, err
	}

	current, err := s.projectRepo.GetActionByID(req.UserID, req.ID)
	if err != nil {
		return domain.GetActionsResponse{}, err
	}
	if err := checkVerb(current, verbBreakDown); err != nil {
		return domain.GetActionsResponse{}, err
	}

	children := make([]domain.Action, len(req.Titles))
	for i, title := range req.Titles {
		children[i] = domain.Action{Title: title, Stage: domain.StageRefine}
	}

	created, err := s.projectRepo.ReplaceWithActions(req.UserID, req.ID, children)
	if err != nil {
		return domain.GetActionsResponse{}, err
	}
	return domain.GetActionsResponse{Actions: created}, nil
}

func (s *service) ReorderActions(req domain.ReorderActionsRequest) (domain.ReorderActionsResponse, error) {
	if err := req.Validate(); err != nil {
		return domain.ReorderActionsResponse{}, err
	}
	if err := s.projectRepo.ReorderActions(req.UserID, req.ProjectID, req.Stage, req.OrderedIDs); err != nil {
		return domain.ReorderActionsResponse{}, err
	}
	return domain.ReorderActionsResponse{ProjectID: req.ProjectID, Stage: req.Stage}, nil
}

func (s *service) DeleteAction(req domain.DeleteActionRequest) (domain.DeleteActionResponse, error) {
	if err := s.projectRepo.DeleteAction(req.UserID, req.ID); err != nil {
		return domain.DeleteActionResponse{}, err
	}
	return domain.DeleteActionResponse{ID: req.ID}, nil
}

// moveToColumn puts an action at the end of a board column, along with any
// extra column updates.
func (s *service) moveToColumn(a domain.Action, to domain.Stage, extra map[string]interface{}) (domain.Action, error) {
	position, err := s.projectRepo.NextPosition(a.UserID, a.ProjectID, to)
	if err != nil {
		return domain.Action{}, err
	}
	updates := map[string]interface{}{"stage": to, "position": position}
	for k, v := range extra {
		updates[k] = v
	}
	return s.applyActionUpdates(a.UserID, a.ID, updates)
}

// applyActionUpdates writes the updates and returns the action as stored, so
// every verb can hand the client its new state.
func (s *service) applyActionUpdates(userID, actionID uint, updates map[string]interface{}) (domain.Action, error) {
	if err := s.projectRepo.UpdateAction(userID, actionID, updates); err != nil {
		return domain.Action{}, err
	}
	return s.projectRepo.GetActionByID(userID, actionID)
}
