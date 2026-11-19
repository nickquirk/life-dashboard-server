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

var wsDate = time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)

// workSessionRepo is a mock holding a single Work session in project 3.
// UpdateWorkSession applies the updates, and any action write fails the test.
func workSessionRepo(t *testing.T, ws domain.WorkSession) (*mocks.MockProjectRepository, *domain.WorkSession) {
	cur := ws
	cur.ID, cur.UserID, cur.ProjectID = 20, 1, 3
	cur.Project = domain.Project{ID: 3, Title: "Garden", Color: "green"}
	failAction := func(userID, actionID uint, updates map[string]interface{}) error {
		t.Fatalf("a Work session write touched action %d", actionID)
		return nil
	}
	repo := &mocks.MockProjectRepository{
		GetWorkSessionByIDFunc: func(userID, id uint) (domain.WorkSession, error) {
			if id != cur.ID || userID != cur.UserID {
				return domain.WorkSession{}, gorm.ErrRecordNotFound
			}
			return cur, nil
		},
		UpdateWorkSessionFunc: func(userID, id uint, updates map[string]interface{}) error {
			if id != cur.ID || userID != cur.UserID {
				return gorm.ErrRecordNotFound
			}
			for k, v := range updates {
				switch k {
				case "date":
					cur.Date = v.(time.Time)
				case "duration_mins":
					cur.DurationMins = v.(int)
				case "completed_at":
					if v == nil {
						cur.CompletedAt = nil
					} else {
						d := v.(time.Time)
						cur.CompletedAt = &d
					}
				}
			}
			return nil
		},
		UpdateActionFunc: failAction,
		UpdateGroupFunc:  failAction,
	}
	return repo, &cur
}

func TestCreateWorkSession(t *testing.T) {
	var saved domain.WorkSession
	group := domain.ActionGroup{ID: 4, Title: "Beds", Color: "pink"}
	repo := &mocks.MockProjectRepository{
		GetProjectByIDFunc: func(userID, projectID uint) (domain.Project, error) {
			return domain.Project{ID: projectID, UserID: userID, Title: "Garden", Color: "green"}, nil
		},
		CreateWorkSessionFunc: func(ws domain.WorkSession) (domain.WorkSession, error) {
			ws.ID = 20
			saved = ws
			return ws, nil
		},
		GetWorkSessionByIDFunc: func(userID, id uint) (domain.WorkSession, error) {
			ws := saved
			ws.Project = domain.Project{ID: 3, Title: "Garden", Color: "green"}
			ws.Group = &group
			return ws, nil
		},
	}
	local := wsDate.In(time.FixedZone("BST", 3600))
	got, err := newProjectService(repo).CreateWorkSession(domain.CreateWorkSessionRequest{
		UserID: 1, ProjectID: 3, GroupID: &group.ID, Date: &local, DurationMins: 60,
	})
	require.NoError(t, err)
	assert.Equal(t, group.ID, *saved.GroupID)
	assert.Equal(t, time.UTC, saved.Date.Location())
	assert.Equal(t, uint(20), got.ID)
	assert.Equal(t, domain.ProjectRef{ID: 3, Title: "Garden", Color: "green"}, got.Project)
	assert.Equal(t, &domain.GroupRef{ID: 4, Title: "Beds"}, got.Group)
}

func TestCreateWorkSession_Refusals(t *testing.T) {
	created := false
	repo := &mocks.MockProjectRepository{
		GetProjectByIDFunc: func(userID, projectID uint) (domain.Project, error) {
			if projectID == 99 {
				return domain.Project{}, gorm.ErrRecordNotFound
			}
			return domain.Project{ID: projectID, IsArchived: projectID == 5}, nil
		},
		CreateWorkSessionFunc: func(ws domain.WorkSession) (domain.WorkSession, error) {
			created = true
			return ws, nil
		},
	}
	svc := newProjectService(repo)

	_, err := svc.CreateWorkSession(domain.CreateWorkSessionRequest{UserID: 1, ProjectID: 5, Date: &wsDate, DurationMins: 60})
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
	assert.Contains(t, err.Error(), "archived")

	_, err = svc.CreateWorkSession(domain.CreateWorkSessionRequest{UserID: 1, ProjectID: 99, Date: &wsDate, DurationMins: 60})
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	_, err = svc.CreateWorkSession(domain.CreateWorkSessionRequest{UserID: 1, ProjectID: 3, DurationMins: 60})
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))

	assert.False(t, created)
}

func TestUpdateWorkSession_MoveAndResize(t *testing.T) {
	done := wsDate
	repo, cur := workSessionRepo(t, domain.WorkSession{Date: wsDate, DurationMins: 60, CompletedAt: &done})
	moved := wsDate.Add(24 * time.Hour)
	got, err := newProjectService(repo).UpdateWorkSession(domain.UpdateWorkSessionRequest{ID: 20, UserID: 1, Date: &moved, DurationMins: ptrInt(90)})
	require.NoError(t, err)
	assert.Equal(t, moved, got.Date)
	assert.Equal(t, 90, got.DurationMins)
	assert.NotNil(t, cur.CompletedAt, "a ticked Work session can still be moved")
	assert.Nil(t, got.Group)

	_, err = newProjectService(repo).UpdateWorkSession(domain.UpdateWorkSessionRequest{ID: 20, UserID: 1})
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
}

func TestCompleteAndReopenWorkSession(t *testing.T) {
	repo, _ := workSessionRepo(t, domain.WorkSession{Date: wsDate, DurationMins: 60})
	svc := newProjectService(repo)

	first, err := svc.CompleteWorkSession(domain.CompleteWorkSessionRequest{ID: 20, UserID: 1})
	require.NoError(t, err)
	require.NotNil(t, first.CompletedAt)

	again, err := svc.CompleteWorkSession(domain.CompleteWorkSessionRequest{ID: 20, UserID: 1})
	require.NoError(t, err)
	assert.Equal(t, *first.CompletedAt, *again.CompletedAt)

	reopened, err := svc.ReopenWorkSession(domain.ReopenWorkSessionRequest{ID: 20, UserID: 1})
	require.NoError(t, err)
	assert.Nil(t, reopened.CompletedAt)

	still, err := svc.ReopenWorkSession(domain.ReopenWorkSessionRequest{ID: 20, UserID: 1})
	require.NoError(t, err)
	assert.Nil(t, still.CompletedAt)

	_, err = svc.CompleteWorkSession(domain.CompleteWorkSessionRequest{ID: 21, UserID: 1})
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestDeleteWorkSession(t *testing.T) {
	var gotID uint
	repo := &mocks.MockProjectRepository{
		DeleteWorkSessionFunc: func(userID, id uint) error { gotID = id; return nil },
	}
	resp, err := newProjectService(repo).DeleteWorkSession(domain.DeleteWorkSessionRequest{ID: 20, UserID: 1})
	require.NoError(t, err)
	assert.Equal(t, uint(20), resp.ID)
	assert.Equal(t, uint(20), gotID)
}

func TestGetWorkSessions_AddsScope(t *testing.T) {
	g := domain.ActionGroup{ID: 4, Title: "Beds"}
	repo := &mocks.MockProjectRepository{
		GetWorkSessionsInRangeFunc: func(userID uint, start, end time.Time) ([]domain.WorkSession, error) {
			p := domain.Project{ID: 3, Title: "Garden", Color: "green"}
			return []domain.WorkSession{{ID: 1, Project: p}, {ID: 2, Project: p, Group: &g}}, nil
		},
	}
	resp, err := newProjectService(repo).GetWorkSessions(domain.GetWorkSessionsRequest{UserID: 1, Start: wsDate, End: wsDate})
	require.NoError(t, err)
	require.Len(t, resp.WorkSessions, 2)
	assert.Nil(t, resp.WorkSessions[0].Group)
	assert.Equal(t, "Beds", resp.WorkSessions[1].Group.Title)
	assert.Equal(t, "green", resp.WorkSessions[1].Project.Color)
}

// --- Ticking actions inside a Work session ---

func TestService_CompleteAction_InWorkSession(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageReady, DurationMins: dur(30)})
	var called []uint
	repo.CompleteActionInWorkSessionFunc = func(userID, actionID, workSessionID uint) (domain.Action, error) {
		called = []uint{userID, actionID, workSessionID}
		return domain.Action{ID: actionID, Stage: domain.StageDone, WorkSessionID: &workSessionID}, nil
	}
	ws := uint(20)

	got, err := newProjectService(repo).CompleteAction(domain.CompleteActionRequest{ID: 10, UserID: 1, WorkSessionID: &ws})
	require.NoError(t, err)
	assert.Equal(t, []uint{1, 10, 20}, called)
	assert.Equal(t, ws, *got.WorkSessionID)
}

func TestService_CompleteAction_InWorkSession_OnlyFromReady(t *testing.T) {
	for _, stage := range []domain.Stage{domain.StageBraindump, domain.StageRefine} {
		t.Run(string(stage), func(t *testing.T) {
			repo, _ := actionRepo(domain.Action{Stage: stage, DurationMins: dur(30)})
			repo.CompleteActionInWorkSessionFunc = func(uint, uint, uint) (domain.Action, error) {
				t.Fatal("should not reach the repository")
				return domain.Action{}, nil
			}
			ws := uint(20)
			_, err := newProjectService(repo).CompleteAction(domain.CompleteActionRequest{ID: 10, UserID: 1, WorkSessionID: &ws})
			requireInvalid(t, err, "Only Ready actions")
		})
	}
}

func TestService_CompleteAction_OnTheBoard_NoLink(t *testing.T) {
	repo, _ := actionRepo(domain.Action{Stage: domain.StageReady, DurationMins: dur(30)})
	repo.CompleteActionInWorkSessionFunc = func(uint, uint, uint) (domain.Action, error) {
		t.Fatal("a board tick should not link")
		return domain.Action{}, nil
	}
	got, err := newProjectService(repo).CompleteAction(domain.CompleteActionRequest{ID: 10, UserID: 1})
	require.NoError(t, err)
	assert.Equal(t, domain.StageDone, got.Stage)
	assert.Nil(t, got.WorkSessionID)
}

func TestService_ReopenAction_ClearsWorkSessionLink(t *testing.T) {
	ws := uint(20)
	repo, _ := actionRepo(domain.Action{Stage: domain.StageDone, DurationMins: dur(30), CompletedAt: at(10), WorkSessionID: &ws})
	got, err := newProjectService(repo).ReopenAction(domain.ReopenActionRequest{ID: 10, UserID: 1})
	require.NoError(t, err)
	assert.Equal(t, domain.StageReady, got.Stage)
	assert.Nil(t, got.WorkSessionID)
}

func TestWithScope_DoneActions(t *testing.T) {
	empty := withScope(domain.WorkSession{})
	assert.NotNil(t, empty.DoneActions)
	assert.Empty(t, empty.DoneActions)

	got := withScope(domain.WorkSession{DoneActions: []domain.Action{{ID: 4, Title: "pot"}, {ID: 5, Title: "soil"}}})
	assert.Equal(t, []domain.ActionRef{{ID: 4, Title: "pot"}, {ID: 5, Title: "soil"}}, got.DoneActions)
}
