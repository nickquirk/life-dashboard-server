package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var wsDay = time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)

func seedWorkSession(t *testing.T, db *gorm.DB, ws domain.WorkSession) domain.WorkSession {
	t.Helper()
	if ws.UserID == 0 {
		ws.UserID = testUserID
	}
	if ws.Date.IsZero() {
		ws.Date = wsDay
	}
	if ws.DurationMins == 0 {
		ws.DurationMins = 60
	}
	require.NoError(t, db.Omit("Project", "Group").Create(&ws).Error)
	return ws
}

func workSessionCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Unscoped().Model(&domain.WorkSession{}).Count(&n).Error)
	return n
}

func TestProjectRepo_CreateWorkSession(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	other := seedProject(t, db, domain.Project{Title: "Other"})
	g := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID})
	foreign := seedGroup(t, db, domain.ActionGroup{ProjectID: other.ID})
	otherUsers := seedGroup(t, db, domain.ActionGroup{UserID: otherProjectUserID, ProjectID: p.ID})

	whole, err := repo.CreateWorkSession(domain.WorkSession{UserID: testUserID, ProjectID: p.ID, Date: wsDay, DurationMins: 60})
	require.NoError(t, err)
	assert.NotZero(t, whole.ID)
	assert.Nil(t, whole.GroupID)

	scoped, err := repo.CreateWorkSession(domain.WorkSession{UserID: testUserID, ProjectID: p.ID, GroupID: &g.ID, Date: wsDay, DurationMins: 30})
	require.NoError(t, err)
	assert.Equal(t, g.ID, *scoped.GroupID)

	for _, bad := range []uint{foreign.ID, otherUsers.ID, 999} {
		_, err := repo.CreateWorkSession(domain.WorkSession{UserID: testUserID, ProjectID: p.ID, GroupID: &bad, Date: wsDay, DurationMins: 30})
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound), "group %d", bad)
	}
	assert.Equal(t, int64(2), workSessionCount(t, db))
}

func TestProjectRepo_GetWorkSessionsInRange(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	archived := seedProject(t, db, domain.Project{Title: "Old", IsArchived: true})
	g := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID, Title: "Beds"})
	start, end := wsDay, wsDay.Add(48*time.Hour)

	late := seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID, Date: end})
	first := seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID, GroupID: &g.ID, Date: start})
	arch := seedWorkSession(t, db, domain.WorkSession{ProjectID: archived.ID, Date: start.Add(time.Hour)})
	seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID, Date: start.Add(-time.Second)})
	seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID, Date: end.Add(time.Second)})
	seedWorkSession(t, db, domain.WorkSession{UserID: otherProjectUserID, ProjectID: p.ID, Date: start})

	got, err := repo.GetWorkSessionsInRange(testUserID, start, end)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, []uint{first.ID, arch.ID, late.ID}, []uint{got[0].ID, got[1].ID, got[2].ID})
	assert.Equal(t, "Garden", got[0].Project.Title)
	require.NotNil(t, got[0].Group)
	assert.Equal(t, "Beds", got[0].Group.Title)
	assert.Nil(t, got[2].Group)
	assert.Equal(t, "Old", got[1].Project.Title)
}

func TestProjectRepo_UpdateWorkSession(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	ws := seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID})
	foreign := seedWorkSession(t, db, domain.WorkSession{UserID: otherProjectUserID, ProjectID: p.ID})

	moved := wsDay.Add(24 * time.Hour)
	require.NoError(t, repo.UpdateWorkSession(testUserID, ws.ID, map[string]interface{}{"date": moved}))
	require.NoError(t, repo.UpdateWorkSession(testUserID, ws.ID, map[string]interface{}{"duration_mins": 90}))
	got, err := repo.GetWorkSessionByID(testUserID, ws.ID)
	require.NoError(t, err)
	assert.True(t, moved.Equal(got.Date))
	assert.Equal(t, 90, got.DurationMins)
	assert.Equal(t, p.ID, got.Project.ID)

	for _, bad := range []uint{foreign.ID, 999} {
		err := repo.UpdateWorkSession(testUserID, bad, map[string]interface{}{"duration_mins": 5})
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound), "id %d", bad)
	}
	_, err = repo.GetWorkSessionByID(testUserID, foreign.ID)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestProjectRepo_DeleteWorkSession(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	ws := seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID})
	foreign := seedWorkSession(t, db, domain.WorkSession{UserID: otherProjectUserID, ProjectID: p.ID})

	require.NoError(t, repo.DeleteWorkSession(testUserID, ws.ID))
	assert.True(t, errors.Is(repo.DeleteWorkSession(testUserID, foreign.ID), gorm.ErrRecordNotFound))
	assert.Equal(t, int64(1), workSessionCount(t, db))
}

func TestProjectRepo_Ungroup_MakesWorkSessionsProjectWide(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	g := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID})
	keep := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID, Title: "Keep"})
	a := seedAction(t, db, domain.Action{ProjectID: p.ID, GroupID: &g.ID})
	past := seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID, GroupID: &g.ID, Date: wsDay.Add(-72 * time.Hour)})
	future := seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID, GroupID: &g.ID})
	untouched := seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID, GroupID: &keep.ID})

	require.NoError(t, repo.Ungroup(testUserID, g.ID))
	for _, id := range []uint{past.ID, future.ID} {
		got, err := repo.GetWorkSessionByID(testUserID, id)
		require.NoError(t, err)
		assert.Nil(t, got.GroupID)
	}
	got, err := repo.GetWorkSessionByID(testUserID, untouched.ID)
	require.NoError(t, err)
	assert.Equal(t, keep.ID, *got.GroupID)
	action, err := repo.GetActionByID(testUserID, a.ID)
	require.NoError(t, err)
	assert.Nil(t, action.GroupID)
}

func TestProjectRepo_DeleteProject_RemovesWorkSessions(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	other := seedProject(t, db, domain.Project{Title: "Other"})
	seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID})
	kept := seedWorkSession(t, db, domain.WorkSession{ProjectID: other.ID})

	require.NoError(t, repo.DeleteProject(testUserID, p.ID))
	assert.Equal(t, int64(1), workSessionCount(t, db))
	_, err := repo.GetWorkSessionByID(testUserID, kept.ID)
	require.NoError(t, err)
}

func TestProjectRepo_CompleteActionInWorkSession(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	other := seedProject(t, db, domain.Project{Title: "Other"})
	g := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID})
	g2 := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID})
	whole := seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID})
	grouped := seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID, GroupID: &g.ID})

	t.Run("links a ticked action and marks it done", func(t *testing.T) {
		a := seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageReady})
		done, err := repo.CompleteActionInWorkSession(testUserID, a.ID, whole.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.StageDone, done.Stage)
		require.NotNil(t, done.CompletedAt)
		require.NotNil(t, done.WorkSessionID)
		assert.Equal(t, whole.ID, *done.WorkSessionID)
	})

	t.Run("a project session takes grouped actions too", func(t *testing.T) {
		a := seedAction(t, db, domain.Action{ProjectID: p.ID, GroupID: &g2.ID, Stage: domain.StageReady})
		_, err := repo.CompleteActionInWorkSession(testUserID, a.ID, whole.ID)
		require.NoError(t, err)
	})

	t.Run("refuses an action outside its scope and changes nothing", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			action  domain.Action
			session uint
		}{
			{"other project", domain.Action{ProjectID: other.ID, Stage: domain.StageReady}, whole.ID},
			{"other group", domain.Action{ProjectID: p.ID, GroupID: &g2.ID, Stage: domain.StageReady}, grouped.ID},
			{"ungrouped in a group session", domain.Action{ProjectID: p.ID, Stage: domain.StageReady}, grouped.ID},
		} {
			a := seedAction(t, db, tc.action)
			_, err := repo.CompleteActionInWorkSession(testUserID, a.ID, tc.session)
			assert.True(t, errors.Is(err, domain.ErrInvalidInput), tc.name)

			var stored domain.Action
			require.NoError(t, db.First(&stored, a.ID).Error)
			assert.Equal(t, domain.StageReady, stored.Stage, tc.name)
			assert.Nil(t, stored.WorkSessionID, tc.name)
		}
	})

	t.Run("an unknown or another user's session is not found", func(t *testing.T) {
		theirs := seedWorkSession(t, db, domain.WorkSession{UserID: otherProjectUserID, ProjectID: p.ID})
		a := seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageReady})
		for _, id := range []uint{999, theirs.ID} {
			_, err := repo.CompleteActionInWorkSession(testUserID, a.ID, id)
			assert.True(t, errors.Is(err, gorm.ErrRecordNotFound), "session %d", id)
		}
	})
}

func TestProjectRepo_WorkSessionReads_IncludeDoneActionsInOrder(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	ws := seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID})
	later, earlier := wsDay.Add(2*time.Hour), wsDay.Add(time.Hour)
	seedAction(t, db, domain.Action{ProjectID: p.ID, Title: "second", Stage: domain.StageDone, CompletedAt: &later, WorkSessionID: &ws.ID})
	seedAction(t, db, domain.Action{ProjectID: p.ID, Title: "first", Stage: domain.StageDone, CompletedAt: &earlier, WorkSessionID: &ws.ID})
	seedAction(t, db, domain.Action{ProjectID: p.ID, Title: "on the board", Stage: domain.StageDone, CompletedAt: &earlier})

	titles := func(actions []domain.Action) []string {
		out := []string{}
		for _, a := range actions {
			out = append(out, a.Title)
		}
		return out
	}

	byID, err := repo.GetWorkSessionByID(testUserID, ws.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, titles(byID.DoneActions))

	inRange, err := repo.GetWorkSessionsInRange(testUserID, wsDay.Add(-time.Hour), wsDay.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, inRange, 1)
	assert.Equal(t, []string{"first", "second"}, titles(inRange[0].DoneActions))
}

func TestProjectRepo_DeleteWorkSession_KeepsActionsDone(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	ws := seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID})
	a := seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageReady})
	_, err := repo.CompleteActionInWorkSession(testUserID, a.ID, ws.ID)
	require.NoError(t, err)
	// A deleted action still holds the foreign key, so its link goes too
	gone := seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageDone, WorkSessionID: &ws.ID})
	require.NoError(t, db.Delete(&gone).Error)

	require.NoError(t, repo.DeleteWorkSession(testUserID, ws.ID))

	var stored domain.Action
	require.NoError(t, db.First(&stored, a.ID).Error)
	assert.Equal(t, domain.StageDone, stored.Stage)
	assert.Nil(t, stored.WorkSessionID)
	var storedGone domain.Action
	require.NoError(t, db.Unscoped().First(&storedGone, gone.ID).Error)
	assert.Nil(t, storedGone.WorkSessionID)
}

func TestProjectRepo_Ungroup_KeepsDoneHereLinks(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	g := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID})
	ws := seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID, GroupID: &g.ID})
	a := seedAction(t, db, domain.Action{ProjectID: p.ID, GroupID: &g.ID, Stage: domain.StageReady})
	_, err := repo.CompleteActionInWorkSession(testUserID, a.ID, ws.ID)
	require.NoError(t, err)

	require.NoError(t, repo.Ungroup(testUserID, g.ID))

	var stored domain.Action
	require.NoError(t, db.First(&stored, a.ID).Error)
	require.NotNil(t, stored.WorkSessionID)
	assert.Equal(t, ws.ID, *stored.WorkSessionID)
}

func TestProjectRepo_DeleteProject_WithDoneHereLinks(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	ws := seedWorkSession(t, db, domain.WorkSession{ProjectID: p.ID})
	a := seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageReady})
	_, err := repo.CompleteActionInWorkSession(testUserID, a.ID, ws.ID)
	require.NoError(t, err)

	require.NoError(t, repo.DeleteProject(testUserID, p.ID))
	assert.Equal(t, int64(0), workSessionCount(t, db))
	assert.Equal(t, int64(0), actionCount(t, db))
}
