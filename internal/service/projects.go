package service

import (
	"fmt"
	"log/slog"

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

	// New actions go to the top of Braindump, where they're easiest to see
	position, err := s.projectRepo.FirstPosition(req.UserID, req.ProjectID, domain.StageBraindump)
	if err != nil {
		return domain.Action{}, err
	}

	// Captured into a completed group, the new action makes it active again
	wasComplete, err := s.groupComplete(req.UserID, req.GroupID)
	if err != nil {
		return domain.Action{}, err
	}

	created, err := s.projectRepo.CreateAction(domain.Action{
		UserID:       req.UserID,
		ProjectID:    req.ProjectID,
		Title:        req.Title,
		Notes:        req.Notes,
		DurationMins: req.DurationMins,
		GroupID:      req.GroupID, // the repository checks it's in this project
		Stage:        domain.StageBraindump,
		Position:     position,
	})
	if err != nil {
		return domain.Action{}, err
	}
	s.recolourIfReactivated(req.UserID, req.GroupID, wasComplete)
	return created, nil
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

	// Reopening an action in a completed group makes the group active again
	wasComplete, err := s.groupComplete(req.UserID, current.GroupID)
	if err != nil {
		return domain.Action{}, err
	}

	var reopened domain.Action
	to := reopenStage(current)
	if to == domain.StageScheduled {
		reopened, err = s.applyActionUpdates(req.UserID, req.ID, map[string]interface{}{"stage": to})
	} else {
		reopened, err = s.moveToColumn(current, to, nil)
	}
	if err != nil {
		return domain.Action{}, err
	}
	s.recolourIfReactivated(req.UserID, current.GroupID, wasComplete)
	return reopened, nil
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

	created, err := s.projectRepo.ReplaceWithActions(req.UserID, req.ID, breakDownChildren(req.Titles))
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
	current, err := s.projectRepo.GetActionByID(req.UserID, req.ID)
	if err != nil {
		return domain.DeleteActionResponse{}, err
	}
	// Deleting the last action of a completed group leaves it empty, and so active
	wasComplete, err := s.groupComplete(req.UserID, current.GroupID)
	if err != nil {
		return domain.DeleteActionResponse{}, err
	}
	if err := s.projectRepo.DeleteAction(req.UserID, req.ID); err != nil {
		return domain.DeleteActionResponse{}, err
	}
	s.recolourIfReactivated(req.UserID, current.GroupID, wasComplete)
	return domain.DeleteActionResponse{ID: req.ID}, nil
}

// breakDownChildren are the new actions a break down creates, in refine.
func breakDownChildren(titles []string) []domain.Action {
	children := make([]domain.Action, len(titles))
	for i, title := range titles {
		children[i] = domain.Action{Title: title, Stage: domain.StageRefine}
	}
	return children
}

// --- Action groups ---
//
// A group lives on one project board, one level deep, and lasts until it's
// ungrouped, even with no actions. An empty group is active.

func (s *service) GetGroups(req domain.GetGroupsRequest) (domain.GetGroupsResponse, error) {
	if _, err := s.projectRepo.GetProjectByID(req.UserID, req.ProjectID); err != nil {
		return domain.GetGroupsResponse{}, err
	}
	groups, err := s.projectRepo.GetGroupsByProjectID(req.UserID, req.ProjectID)
	if err != nil {
		return domain.GetGroupsResponse{}, err
	}
	return domain.GetGroupsResponse{Groups: groups}, nil
}

// NewGroup makes an empty group on the board, coloured like a new one from
// Create group.
func (s *service) NewGroup(req domain.NewGroupRequest) (domain.ActionGroup, error) {
	if err := req.Validate(); err != nil {
		return domain.ActionGroup{}, err
	}
	// Also checks the project is the user's
	color, err := s.pickGroupColor(req.UserID, req.ProjectID, 0)
	if err != nil {
		return domain.ActionGroup{}, err
	}
	return s.projectRepo.CreateEmptyGroup(domain.ActionGroup{
		UserID: req.UserID, ProjectID: req.ProjectID, Title: req.Title, Color: color,
	})
}

// CreateGroup breaks an action down into a new group named after it.
func (s *service) CreateGroup(req domain.CreateGroupRequest) (domain.CreateGroupResponse, error) {
	if err := req.Validate(); err != nil {
		return domain.CreateGroupResponse{}, err
	}

	current, err := s.projectRepo.GetActionByID(req.UserID, req.ID)
	if err != nil {
		return domain.CreateGroupResponse{}, err
	}
	if err := checkVerb(current, verbBreakDown); err != nil {
		return domain.CreateGroupResponse{}, err
	}
	if current.GroupID != nil {
		return domain.CreateGroupResponse{}, fmt.Errorf("%w: this action is already in a group; use Break down", domain.ErrInvalidInput)
	}

	if req.Color == "" {
		if req.Color, err = s.pickGroupColor(req.UserID, current.ProjectID, 0); err != nil {
			return domain.CreateGroupResponse{}, err
		}
	}

	group, created, err := s.projectRepo.CreateGroupFromAction(req.UserID, req.ID,
		domain.ActionGroup{Title: current.Title, Color: req.Color}, breakDownChildren(req.Titles))
	if err != nil {
		return domain.CreateGroupResponse{}, err
	}
	return domain.CreateGroupResponse{Group: group, Actions: created}, nil
}

// pickGroupColor avoids the project's swatch and its other active groups'
// where possible, leaving out exceptGroupID (0 for a new group). Completed
// groups have given their colour back, so they don't count.
func (s *service) pickGroupColor(userID, projectID, exceptGroupID uint) (string, error) {
	project, err := s.projectRepo.GetProjectByID(userID, projectID)
	if err != nil {
		return "", err
	}
	active, err := s.projectRepo.ActiveGroupColors(userID, projectID, exceptGroupID)
	if err != nil {
		return "", err
	}
	return domain.PickSwatch(append([]string{project.Color}, active...)), nil
}

// groupComplete reports whether groupID is a completed group. No group is
// never complete.
func (s *service) groupComplete(userID uint, groupID *uint) (bool, error) {
	if groupID == nil {
		return false, nil
	}
	return s.projectRepo.GroupIsComplete(userID, *groupID)
}

// recolourIfReactivated gives a group that was completed before a write and
// is active after it a fresh colour: only active groups hold one, so its old
// colour may have gone to another group since. Called after the write has
// succeeded, so a failure here only leaves the old colour, and is logged
// rather than failing the request.
func (s *service) recolourIfReactivated(userID uint, groupID *uint, wasComplete bool) {
	if groupID == nil || !wasComplete {
		return
	}
	if err := s.recolourGroup(userID, *groupID); err != nil {
		slog.Error("failed to recolour reactivated group", "error", err, "userID", userID, "groupID", *groupID)
	}
}

func (s *service) recolourGroup(userID, groupID uint) error {
	complete, err := s.projectRepo.GroupIsComplete(userID, groupID)
	if err != nil || complete {
		return err
	}
	group, err := s.projectRepo.GetGroupByID(userID, groupID)
	// A pinned group never gave its colour back
	if err != nil || group.IsPinned {
		return err
	}
	color, err := s.pickGroupColor(userID, group.ProjectID, group.ID)
	if err != nil {
		return err
	}
	return s.projectRepo.UpdateGroup(userID, group.ID, map[string]interface{}{"color": color})
}

// SetActionGroup moves an action into a group on its own board, or out of its
// group when GroupID is nil. Allowed in every stage.
func (s *service) SetActionGroup(req domain.SetActionGroupRequest) (domain.Action, error) {
	if err := req.Validate(); err != nil {
		return domain.Action{}, err
	}

	current, err := s.projectRepo.GetActionByID(req.UserID, req.ID)
	if err != nil {
		return domain.Action{}, err
	}
	if sameGroup(current.GroupID, req.GroupID) {
		return current, nil
	}

	// Moving an unfinished action into a completed group makes it active
	// again, and so does moving the last action out of one (it's now empty)
	wasComplete, err := s.groupComplete(req.UserID, req.GroupID)
	if err != nil {
		return domain.Action{}, err
	}
	leftWasComplete, err := s.groupComplete(req.UserID, current.GroupID)
	if err != nil {
		return domain.Action{}, err
	}

	// The repository checks the group is on this action's board, in its transaction.
	moved, err := s.projectRepo.SetActionGroup(req.UserID, req.ID, req.GroupID)
	if err != nil {
		return domain.Action{}, err
	}
	s.recolourIfReactivated(req.UserID, req.GroupID, wasComplete)
	s.recolourIfReactivated(req.UserID, current.GroupID, leftWasComplete)
	return moved, nil
}

// colorForPinning is a fresh colour for an unpinned completed group about to
// be pinned, or "" when the group keeps the one it holds.
func (s *service) colorForPinning(userID, groupID uint) (string, error) {
	group, err := s.projectRepo.GetGroupByID(userID, groupID)
	if err != nil || group.IsPinned {
		return "", err
	}
	complete, err := s.projectRepo.GroupIsComplete(userID, groupID)
	if err != nil || !complete {
		return "", err
	}
	return s.pickGroupColor(userID, group.ProjectID, group.ID)
}

func sameGroup(a, b *uint) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func (s *service) UpdateGroup(req domain.UpdateGroupRequest) (domain.ActionGroup, error) {
	if err := req.Validate(); err != nil {
		return domain.ActionGroup{}, err
	}

	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Color != nil {
		updates["color"] = *req.Color
	}
	if req.IsPinned != nil {
		updates["is_pinned"] = *req.IsPinned
		if *req.IsPinned && req.Color == nil {
			// A completed group has given its colour back, and a newer group
			// may have it now, so pinning one picks a fresh colour
			color, err := s.colorForPinning(req.UserID, req.ID)
			if err != nil {
				return domain.ActionGroup{}, err
			}
			if color != "" {
				updates["color"] = color
			}
		}
	}
	if err := s.projectRepo.UpdateGroup(req.UserID, req.ID, updates); err != nil {
		return domain.ActionGroup{}, err
	}
	return s.projectRepo.GetGroupByID(req.UserID, req.ID)
}

func (s *service) Ungroup(req domain.UngroupRequest) (domain.UngroupResponse, error) {
	if err := s.projectRepo.Ungroup(req.UserID, req.ID); err != nil {
		return domain.UngroupResponse{}, err
	}
	return domain.UngroupResponse{ID: req.ID}, nil
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
