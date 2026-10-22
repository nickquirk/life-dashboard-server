package service

import (
	"errors"
	"testing"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const nextPos = 7

func newProjectService(repo *mocks.MockProjectRepository) Service {
	return NewServiceWithRepos(nil, nil, nil, nil, nil, nil, nil, nil, nil, repo)
}

// actionRepo is a mock holding a single action. UpdateAction applies the
// updates to it, so a verb's returned action reflects what was written.
func actionRepo(a domain.Action) (*mocks.MockProjectRepository, *domain.Action) {
	cur := a
	cur.ID, cur.UserID, cur.ProjectID = 10, 1, 3
	repo := &mocks.MockProjectRepository{
		GetActionByIDFunc: func(userID, actionID uint) (domain.Action, error) {
			if actionID != cur.ID || userID != cur.UserID {
				return domain.Action{}, gorm.ErrRecordNotFound
			}
			return cur, nil
		},
		NextPositionFunc: func(userID, projectID uint, stage domain.Stage) (int, error) {
			return nextPos, nil
		},
		UpdateActionFunc: func(userID, actionID uint, updates map[string]interface{}) error {
			for k, v := range updates {
				switch k {
				case "stage":
					cur.Stage = v.(domain.Stage)
				case "position":
					cur.Position = v.(int)
				case "title":
					cur.Title = v.(string)
				case "notes":
					cur.Notes = v.(string)
				case "duration_mins":
					cur.DurationMins = v.(*int)
				case "date":
					if v == nil {
						cur.Date = nil
					} else {
						d := v.(time.Time)
						cur.Date = &d
					}
				}
			}
			return nil
		},
	}
	return repo, &cur
}

func dur(m int) *int { return &m }

func at(h int) *time.Time {
	t := time.Date(2026, 10, 17, h, 0, 0, 0, time.UTC)
	return &t
}

func requireInvalid(t *testing.T, err error, contains string) {
	t.Helper()
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput), "want ErrInvalidInput, got %v", err)
	assert.Contains(t, err.Error(), contains)
}

// --- ChangeActionStage ---

func TestService_ChangeStage_BoardMovesAppendToColumn(t *testing.T) {
	cases := []struct {
		from, to domain.Stage
		duration *int
	}{
		{domain.StageBraindump, domain.StageRefine, nil},
		{domain.StageRefine, domain.StageBraindump, nil},
		{domain.StageRefine, domain.StageReady, dur(30)},
		{domain.StageBraindump, domain.StageReady, dur(30)},
		{domain.StageReady, domain.StageRefine, dur(30)},
		{domain.StageReady, domain.StageBraindump, dur(30)},
	}
	for _, c := range cases {
		t.Run(string(c.from)+"->"+string(c.to), func(t *testing.T) {
			repo, _ := actionRepo(domain.Action{Stage: c.from, DurationMins: c.duration})
			got, err := newProjectService(repo).ChangeActionStage(domain.ChangeStageRequest{ID: 10, UserID: 1, Stage: c.to})
			require.NoError(t, err)
			assert.Equal(t, c.to, got.Stage)
			assert.Equal(t, nextPos, got.Position)
			assert.Nil(t, got.Date)
		})
	}
}

func TestService_ChangeStage_ReadyRequiresDuration(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageRefine})
	_, err := newProjectService(repo).ChangeActionStage(domain.ChangeStageRequest{ID: 10, UserID: 1, Stage: domain.StageReady})
	requireInvalid(t, err, "duration")
}

func TestService_ChangeStage_RejectedFromScheduledAndDone(t *testing.T) {
	for _, from := range []domain.Stage{domain.StageScheduled, domain.StageDone} {
		repo, _ := actionRepo(domain.Action{Stage: from, DurationMins: dur(30), Date: at(9)})
		_, err := newProjectService(repo).ChangeActionStage(domain.ChangeStageRequest{ID: 10, UserID: 1, Stage: domain.StageRefine})
		requireInvalid(t, err, string(from))
	}
}

func TestService_ChangeStage_RejectsNonBoardTarget(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageReady, DurationMins: dur(30)})
	_, err := newProjectService(repo).ChangeActionStage(domain.ChangeStageRequest{ID: 10, UserID: 1, Stage: domain.StageScheduled})
	requireInvalid(t, err, "stage")
}

func TestService_ChangeStage_SameStageRejected(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageRefine})
	_, err := newProjectService(repo).ChangeActionStage(domain.ChangeStageRequest{ID: 10, UserID: 1, Stage: domain.StageRefine})
	requireInvalid(t, err, "already")
}

func TestService_ChangeStage_NotFound(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageRefine})
	_, err := newProjectService(repo).ChangeActionStage(domain.ChangeStageRequest{ID: 99, UserID: 1, Stage: domain.StageBraindump})
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

// --- ScheduleAction ---

func TestService_Schedule_FromReady(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageReady, DurationMins: dur(30)})
	local := time.Date(2026, 10, 17, 9, 0, 0, 0, time.FixedZone("BST", 3600))
	got, err := newProjectService(repo).ScheduleAction(domain.ScheduleActionRequest{ID: 10, UserID: 1, Date: &local})
	require.NoError(t, err)
	assert.Equal(t, domain.StageScheduled, got.Stage)
	require.NotNil(t, got.Date)
	assert.True(t, got.Date.Equal(local))
	assert.Equal(t, time.UTC, got.Date.Location(), "stored as UTC")
}

func TestService_Schedule_RejectedOutsideReady(t *testing.T) {
	for _, from := range []domain.Stage{domain.StageBraindump, domain.StageRefine, domain.StageScheduled, domain.StageDone} {
		repo, _ := actionRepo(domain.Action{Stage: from, DurationMins: dur(30)})
		_, err := newProjectService(repo).ScheduleAction(domain.ScheduleActionRequest{ID: 10, UserID: 1, Date: at(9)})
		requireInvalid(t, err, "schedule")
	}
}

func TestService_Schedule_RequiresDate(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageReady, DurationMins: dur(30)})
	_, err := newProjectService(repo).ScheduleAction(domain.ScheduleActionRequest{ID: 10, UserID: 1})
	requireInvalid(t, err, "date")
}

// --- MoveAction ---

func TestService_Move_ChangesDate(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageScheduled, DurationMins: dur(30), Date: at(9)})
	got, err := newProjectService(repo).MoveAction(domain.MoveActionRequest{ID: 10, UserID: 1, Date: at(14)})
	require.NoError(t, err)
	assert.Equal(t, domain.StageScheduled, got.Stage)
	assert.True(t, got.Date.Equal(*at(14)))
}

func TestService_Move_RejectedUnlessScheduled(t *testing.T) {
	for _, from := range []domain.Stage{domain.StageReady, domain.StageDone} {
		repo, _ := actionRepo(domain.Action{Stage: from, DurationMins: dur(30)})
		_, err := newProjectService(repo).MoveAction(domain.MoveActionRequest{ID: 10, UserID: 1, Date: at(14)})
		requireInvalid(t, err, "move")
	}
}

// --- UnscheduleAction ---

func TestService_Unschedule_BackToReady(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageScheduled, DurationMins: dur(30), Date: at(9)})
	got, err := newProjectService(repo).UnscheduleAction(domain.UnscheduleActionRequest{ID: 10, UserID: 1})
	require.NoError(t, err)
	assert.Equal(t, domain.StageReady, got.Stage)
	assert.Nil(t, got.Date)
	assert.Equal(t, nextPos, got.Position)
}

func TestService_Unschedule_RejectedUnlessScheduled(t *testing.T) {
	for _, from := range []domain.Stage{domain.StageReady, domain.StageDone} {
		repo, _ := actionRepo(domain.Action{Stage: from, DurationMins: dur(30)})
		_, err := newProjectService(repo).UnscheduleAction(domain.UnscheduleActionRequest{ID: 10, UserID: 1})
		requireInvalid(t, err, "unschedule")
	}
}

// --- CompleteAction ---

func TestService_Complete_FromBoardStages_StaysUndated(t *testing.T) {
	for _, from := range []domain.Stage{domain.StageBraindump, domain.StageRefine, domain.StageReady} {
		repo, _ := actionRepo(domain.Action{Stage: from})
		got, err := newProjectService(repo).CompleteAction(domain.CompleteActionRequest{ID: 10, UserID: 1})
		require.NoError(t, err)
		assert.Equal(t, domain.StageDone, got.Stage)
		assert.Nil(t, got.Date)
	}
}

func TestService_Complete_FromScheduled_KeepsDate(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageScheduled, DurationMins: dur(30), Date: at(9)})
	got, err := newProjectService(repo).CompleteAction(domain.CompleteActionRequest{ID: 10, UserID: 1})
	require.NoError(t, err)
	assert.Equal(t, domain.StageDone, got.Stage)
	require.NotNil(t, got.Date)
	assert.True(t, got.Date.Equal(*at(9)))
}

func TestService_Complete_AlreadyDoneRejected(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageDone})
	_, err := newProjectService(repo).CompleteAction(domain.CompleteActionRequest{ID: 10, UserID: 1})
	requireInvalid(t, err, "complete")
}

// --- ReopenAction ---

func TestService_Reopen_Outcomes(t *testing.T) {
	cases := []struct {
		name     string
		action   domain.Action
		want     domain.Stage
		wantDate bool
	}{
		{"with date -> scheduled", domain.Action{Stage: domain.StageDone, DurationMins: dur(30), Date: at(9)}, domain.StageScheduled, true},
		{"duration only -> ready", domain.Action{Stage: domain.StageDone, DurationMins: dur(30)}, domain.StageReady, false},
		{"neither -> refine", domain.Action{Stage: domain.StageDone}, domain.StageRefine, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, _ := actionRepo(c.action)
			got, err := newProjectService(repo).ReopenAction(domain.ReopenActionRequest{ID: 10, UserID: 1})
			require.NoError(t, err)
			assert.Equal(t, c.want, got.Stage)
			assert.Equal(t, c.wantDate, got.Date != nil)
		})
	}
}

func TestService_Reopen_RejectedUnlessDone(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageReady, DurationMins: dur(30)})
	_, err := newProjectService(repo).ReopenAction(domain.ReopenActionRequest{ID: 10, UserID: 1})
	requireInvalid(t, err, "reopen")
}

// --- BreakDownAction ---

func TestService_BreakDown_ChildrenLandInRefine(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageBraindump})
	var gotChildren []domain.Action
	repo.ReplaceWithActionsFunc = func(userID, actionID uint, children []domain.Action) ([]domain.Action, error) {
		gotChildren = children
		return children, nil
	}

	resp, err := newProjectService(repo).BreakDownAction(domain.BreakDownActionRequest{
		ID: 10, UserID: 1, Titles: []string{" measure the space ", "buy timber"},
	})
	require.NoError(t, err)
	require.Len(t, gotChildren, 2)
	assert.Equal(t, "measure the space", gotChildren[0].Title)
	assert.Equal(t, "buy timber", gotChildren[1].Title)
	for _, c := range gotChildren {
		assert.Equal(t, domain.StageRefine, c.Stage)
		assert.Nil(t, c.DurationMins)
	}
	assert.Len(t, resp.Actions, 2)
}

func TestService_BreakDown_RejectedFromScheduledAndDone(t *testing.T) {
	for _, from := range []domain.Stage{domain.StageScheduled, domain.StageDone} {
		repo, _ := actionRepo(domain.Action{Stage: from, DurationMins: dur(30), Date: at(9)})
		called := false
		repo.ReplaceWithActionsFunc = func(uint, uint, []domain.Action) ([]domain.Action, error) {
			called = true
			return nil, nil
		}
		_, err := newProjectService(repo).BreakDownAction(domain.BreakDownActionRequest{ID: 10, UserID: 1, Titles: []string{"a", "b"}})
		requireInvalid(t, err, "break down")
		assert.False(t, called)
	}
}

func TestService_BreakDown_NeedsTwoTitles(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageRefine})
	_, err := newProjectService(repo).BreakDownAction(domain.BreakDownActionRequest{ID: 10, UserID: 1, Titles: []string{"a"}})
	requireInvalid(t, err, "at least")
}

// --- UpdateAction ---

func TestService_UpdateAction_ClearingDurationRejectedWhenNeeded(t *testing.T) {
	for _, from := range []domain.Stage{domain.StageReady, domain.StageScheduled} {
		repo, _ := actionRepo(domain.Action{Stage: from, DurationMins: dur(30)})
		_, err := newProjectService(repo).UpdateAction(domain.UpdateActionRequest{
			ID: 10, UserID: 1, DurationMins: domain.NullableInt{Set: true},
		})
		requireInvalid(t, err, "duration")
	}
}

func TestService_UpdateAction_ClearingDurationAllowedElsewhere(t *testing.T) {
	for _, from := range []domain.Stage{domain.StageBraindump, domain.StageRefine, domain.StageDone} {
		repo, _ := actionRepo(domain.Action{Stage: from, DurationMins: dur(30)})
		got, err := newProjectService(repo).UpdateAction(domain.UpdateActionRequest{
			ID: 10, UserID: 1, DurationMins: domain.NullableInt{Set: true},
		})
		require.NoError(t, err)
		assert.Nil(t, got.DurationMins)
	}
}

func TestService_UpdateAction_OmittedDurationUntouched(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageReady, DurationMins: dur(30), Title: "old"})
	var captured map[string]interface{}
	inner := repo.UpdateActionFunc
	repo.UpdateActionFunc = func(userID, actionID uint, updates map[string]interface{}) error {
		captured = updates
		return inner(userID, actionID, updates)
	}
	got, err := newProjectService(repo).UpdateAction(domain.UpdateActionRequest{ID: 10, UserID: 1, Title: ptrStr("new")})
	require.NoError(t, err)
	assert.Equal(t, "new", got.Title)
	assert.NotContains(t, captured, "duration_mins")
	require.NotNil(t, got.DurationMins)
	assert.Equal(t, 30, *got.DurationMins)
}

func TestService_UpdateAction_NoChangesReturnsCurrent(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageRefine, Title: "x"})
	repo.UpdateActionFunc = func(uint, uint, map[string]interface{}) error {
		t.Fatal("UpdateAction should not be called")
		return nil
	}
	got, err := newProjectService(repo).UpdateAction(domain.UpdateActionRequest{ID: 10, UserID: 1})
	require.NoError(t, err)
	assert.Equal(t, "x", got.Title)
}

// --- CreateAction ---

func TestService_CreateAction_LandsAtTopOfBraindump(t *testing.T) {
	var captured domain.Action
	repo := &mocks.MockProjectRepository{
		GetProjectByIDFunc: func(userID, projectID uint) (domain.Project, error) {
			return domain.Project{ID: projectID, UserID: userID}, nil
		},
		FirstPositionFunc: func(userID, projectID uint, stage domain.Stage) (int, error) {
			assert.Equal(t, domain.StageBraindump, stage)
			return -2, nil
		},
		CreateActionFunc: func(a domain.Action) (domain.Action, error) {
			captured = a
			a.ID = 11
			return a, nil
		},
	}
	got, err := newProjectService(repo).CreateAction(domain.CreateActionRequest{UserID: 1, ProjectID: 3, Title: " buy compost "})
	require.NoError(t, err)
	assert.Equal(t, uint(11), got.ID)
	assert.Equal(t, domain.StageBraindump, captured.Stage)
	assert.Equal(t, -2, captured.Position)
	assert.Equal(t, "buy compost", captured.Title)
	assert.Equal(t, uint(3), captured.ProjectID)
}

func TestService_CreateAction_ArchivedProjectRejected(t *testing.T) {
	repo := &mocks.MockProjectRepository{
		GetProjectByIDFunc: func(userID, projectID uint) (domain.Project, error) {
			return domain.Project{ID: projectID, IsArchived: true}, nil
		},
		CreateActionFunc: func(a domain.Action) (domain.Action, error) {
			t.Fatal("CreateAction should not be called")
			return a, nil
		},
	}
	_, err := newProjectService(repo).CreateAction(domain.CreateActionRequest{UserID: 1, ProjectID: 3, Title: "x"})
	requireInvalid(t, err, "archived")
}

func TestService_CreateAction_UnknownProject(t *testing.T) {
	repo := &mocks.MockProjectRepository{
		GetProjectByIDFunc: func(userID, projectID uint) (domain.Project, error) {
			return domain.Project{}, gorm.ErrRecordNotFound
		},
	}
	_, err := newProjectService(repo).CreateAction(domain.CreateActionRequest{UserID: 1, ProjectID: 3, Title: "x"})
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

// --- Reads ---

func TestService_GetScheduledActions_TrimsProject(t *testing.T) {
	repo := &mocks.MockProjectRepository{
		GetActionsInRangeFunc: func(userID uint, start, end time.Time) ([]domain.Action, error) {
			return []domain.Action{{
				ID: 1, Stage: domain.StageScheduled, Date: at(9),
				Project: domain.Project{ID: 3, Title: "Garden", Color: "green", UserID: 1},
			}}, nil
		},
	}
	resp, err := newProjectService(repo).GetScheduledActions(domain.GetScheduledActionsRequest{UserID: 1})
	require.NoError(t, err)
	require.Len(t, resp.Actions, 1)
	assert.Equal(t, domain.ProjectRef{ID: 3, Title: "Garden", Color: "green"}, resp.Actions[0].Project)
}

func TestService_GetReadyActions_EmptyIsNonNil(t *testing.T) {
	repo := &mocks.MockProjectRepository{
		GetReadyActionsFunc: func(userID uint) ([]domain.Action, error) { return []domain.Action{}, nil },
	}
	resp, err := newProjectService(repo).GetReadyActions(domain.GetReadyActionsRequest{UserID: 1})
	require.NoError(t, err)
	assert.NotNil(t, resp.Actions)
}

// --- Projects ---

func TestService_CreateProject_Validates(t *testing.T) {
	repo := &mocks.MockProjectRepository{
		CreateProjectFunc: func(p domain.Project) (domain.Project, error) {
			t.Fatal("CreateProject should not be called")
			return p, nil
		},
	}
	_, err := newProjectService(repo).CreateProject(domain.CreateProjectRequest{UserID: 1, Title: "Garden"})
	requireInvalid(t, err, "color")
}

func TestService_UpdateProject_ArchiveReturnsUpdated(t *testing.T) {
	var captured map[string]interface{}
	repo := &mocks.MockProjectRepository{
		UpdateProjectFunc: func(userID, projectID uint, updates map[string]interface{}) error {
			captured = updates
			return nil
		},
		GetProjectByIDFunc: func(userID, projectID uint) (domain.Project, error) {
			return domain.Project{ID: projectID, IsArchived: true}, nil
		},
	}
	got, err := newProjectService(repo).UpdateProject(domain.UpdateProjectRequest{ID: 3, UserID: 1, IsArchived: ptrBool(true)})
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"is_archived": true}, captured)
	assert.True(t, got.IsArchived)
}

func TestService_UpdateProject_NotFound(t *testing.T) {
	repo := &mocks.MockProjectRepository{
		UpdateProjectFunc: func(uint, uint, map[string]interface{}) error { return gorm.ErrRecordNotFound },
	}
	_, err := newProjectService(repo).UpdateProject(domain.UpdateProjectRequest{ID: 3, UserID: 1, Title: ptrStr("x")})
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}
