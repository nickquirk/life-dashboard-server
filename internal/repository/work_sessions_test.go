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
