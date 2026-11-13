package repository

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const otherProjectUserID uint = 2

func newProjectRepo(t *testing.T) (*GormProjectRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.Project{}, &domain.Action{}, &domain.ActionGroup{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return &GormProjectRepository{Db: db}, db
}

func seedProject(t *testing.T, db *gorm.DB, p domain.Project) domain.Project {
	t.Helper()
	if p.UserID == 0 {
		p.UserID = testUserID
	}
	if p.Title == "" {
		p.Title = "Garden"
	}
	if p.Color == "" {
		p.Color = "green"
	}
	require.NoError(t, db.Create(&p).Error)
	return p
}

func seedAction(t *testing.T, db *gorm.DB, a domain.Action) domain.Action {
	t.Helper()
	if a.UserID == 0 {
		a.UserID = testUserID
	}
	if a.Title == "" {
		a.Title = "buy compost"
	}
	if a.Stage == "" {
		a.Stage = domain.StageBraindump
	}
	require.NoError(t, db.Omit("Project").Create(&a).Error)
	return a
}

func actionCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Unscoped().Model(&domain.Action{}).Count(&n).Error)
	return n
}

// --- Projects ---

func TestProjectRepo_GetProjects_Stats(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	seedAction(t, db, domain.Action{ProjectID: p.ID})
	seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageDone})
	seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageDone})
	empty := seedProject(t, db, domain.Project{Title: "Empty"})

	projects, err := repo.GetProjectsByUserID(testUserID, false)
	require.NoError(t, err)
	require.Len(t, projects, 2)
	assert.Equal(t, p.ID, projects[0].ID)
	assert.Equal(t, 3, projects[0].ActionCount)
	assert.Equal(t, 2, projects[0].DoneCount)
	assert.Equal(t, empty.ID, projects[1].ID)
	assert.Equal(t, 0, projects[1].ActionCount)
	assert.Equal(t, 0, projects[1].DoneCount)
}

func TestProjectRepo_GetProjects_FiltersArchivedAndUser(t *testing.T) {
	repo, db := newProjectRepo(t)
	active := seedProject(t, db, domain.Project{})
	archived := seedProject(t, db, domain.Project{IsArchived: true})
	seedProject(t, db, domain.Project{UserID: otherProjectUserID})

	got, err := repo.GetProjectsByUserID(testUserID, false)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, active.ID, got[0].ID)

	got, err = repo.GetProjectsByUserID(testUserID, true)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, archived.ID, got[0].ID)
}

func TestProjectRepo_GetProjects_EmptyIsNonNil(t *testing.T) {
	repo, _ := newProjectRepo(t)
	got, err := repo.GetProjectsByUserID(testUserID, false)
	require.NoError(t, err)
	b, _ := json.Marshal(got)
	assert.Equal(t, "[]", string(b))
}

func TestProjectRepo_GetProjectByID_WrongUser(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{UserID: otherProjectUserID})
	_, err := repo.GetProjectByID(testUserID, p.ID)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestProjectRepo_UpdateProject_WrongUser(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{UserID: otherProjectUserID})
	err := repo.UpdateProject(testUserID, p.ID, map[string]interface{}{"title": "x"})
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestProjectRepo_DeleteProject_RemovesItsActions(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	other := seedProject(t, db, domain.Project{Title: "Other"})
	date := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	seedAction(t, db, domain.Action{ProjectID: p.ID})
	seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageScheduled, Date: &date, DurationMins: ptrInt(30)})
	kept := seedAction(t, db, domain.Action{ProjectID: other.ID})

	require.NoError(t, repo.DeleteProject(testUserID, p.ID))

	assert.Equal(t, int64(1), actionCount(t, db))
	_, err := repo.GetActionByID(testUserID, kept.ID)
	assert.NoError(t, err)
	inRange, err := repo.GetActionsInRange(testUserID, date.Add(-time.Hour), date.Add(time.Hour))
	require.NoError(t, err)
	assert.Empty(t, inRange)
}

func TestProjectRepo_DeleteProject_WrongUserChangesNothing(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{UserID: otherProjectUserID})
	seedAction(t, db, domain.Action{UserID: otherProjectUserID, ProjectID: p.ID})

	err := repo.DeleteProject(testUserID, p.ID)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	assert.Equal(t, int64(1), actionCount(t, db))
}

// --- Actions ---

func TestProjectRepo_CreateAction_ForeignProject(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{UserID: otherProjectUserID})
	_, err := repo.CreateAction(domain.Action{UserID: testUserID, ProjectID: p.ID, Title: "x", Stage: domain.StageBraindump})
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	assert.Equal(t, int64(0), actionCount(t, db))
}

func TestProjectRepo_GetActionByID_WrongUser(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{UserID: otherProjectUserID})
	a := seedAction(t, db, domain.Action{UserID: otherProjectUserID, ProjectID: p.ID})
	_, err := repo.GetActionByID(testUserID, a.ID)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestProjectRepo_UpdateAction_WrongUser(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{UserID: otherProjectUserID})
	a := seedAction(t, db, domain.Action{UserID: otherProjectUserID, ProjectID: p.ID})
	err := repo.UpdateAction(testUserID, a.ID, map[string]interface{}{"title": "x"})
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestProjectRepo_UpdateAction_ClearsDate(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	date := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	a := seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageScheduled, Date: &date})

	require.NoError(t, repo.UpdateAction(testUserID, a.ID, map[string]interface{}{"stage": domain.StageReady, "date": nil}))
	got, err := repo.GetActionByID(testUserID, a.ID)
	require.NoError(t, err)
	assert.Nil(t, got.Date)
	assert.Equal(t, domain.StageReady, got.Stage)
}

func TestProjectRepo_GetActionsByProjectID_OrderAndScope(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	second := seedAction(t, db, domain.Action{ProjectID: p.ID, Position: 1})
	first := seedAction(t, db, domain.Action{ProjectID: p.ID, Position: 0})
	other := seedProject(t, db, domain.Project{Title: "Other"})
	seedAction(t, db, domain.Action{ProjectID: other.ID})

	got, err := repo.GetActionsByProjectID(testUserID, p.ID)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, first.ID, got[0].ID)
	assert.Equal(t, second.ID, got[1].ID)
}

func TestProjectRepo_GetActionsByProjectID_ForeignProject(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{UserID: otherProjectUserID})
	_, err := repo.GetActionsByProjectID(testUserID, p.ID)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestProjectRepo_GetActionsByProjectID_EmptyIsNonNil(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	got, err := repo.GetActionsByProjectID(testUserID, p.ID)
	require.NoError(t, err)
	assert.NotNil(t, got)
}

func TestProjectRepo_GetActionsInRange_BoundariesAndProject(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{Title: "Garden", Color: "green"})
	archived := seedProject(t, db, domain.Project{Title: "Old", Color: "pink", IsArchived: true})
	start := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 18, 23, 59, 59, 0, time.UTC)
	before := start.Add(-time.Second)
	after := end.Add(time.Second)
	onStart, onEnd := start, end

	seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageScheduled, Date: &before})
	a1 := seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageScheduled, Date: &onStart})
	a2 := seedAction(t, db, domain.Action{ProjectID: archived.ID, Stage: domain.StageDone, Date: &onEnd})
	seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageScheduled, Date: &after})
	seedAction(t, db, domain.Action{ProjectID: p.ID}) // undated
	otherP := seedProject(t, db, domain.Project{UserID: otherProjectUserID})
	seedAction(t, db, domain.Action{UserID: otherProjectUserID, ProjectID: otherP.ID, Stage: domain.StageScheduled, Date: &onStart})

	got, err := repo.GetActionsInRange(testUserID, start, end)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, a1.ID, got[0].ID)
	assert.Equal(t, "Garden", got[0].Project.Title)
	assert.Equal(t, a2.ID, got[1].ID, "archived projects' actions stay on the calendar")
	assert.Equal(t, "pink", got[1].Project.Color)
}

func TestProjectRepo_GetReadyActions_SkipsArchivedAndOtherStages(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	archived := seedProject(t, db, domain.Project{IsArchived: true})
	ready := seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageReady, DurationMins: ptrInt(30)})
	seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageRefine})
	seedAction(t, db, domain.Action{ProjectID: archived.ID, Stage: domain.StageReady, DurationMins: ptrInt(30)})
	otherP := seedProject(t, db, domain.Project{UserID: otherProjectUserID})
	seedAction(t, db, domain.Action{UserID: otherProjectUserID, ProjectID: otherP.ID, Stage: domain.StageReady})

	got, err := repo.GetReadyActions(testUserID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, ready.ID, got[0].ID)
	assert.Equal(t, p.Title, got[0].Project.Title)
}

func TestProjectRepo_NextPosition(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})

	pos, err := repo.NextPosition(testUserID, p.ID, domain.StageRefine)
	require.NoError(t, err)
	assert.Equal(t, 0, pos)

	seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageRefine, Position: 4})
	seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageBraindump, Position: 9})
	pos, err = repo.NextPosition(testUserID, p.ID, domain.StageRefine)
	require.NoError(t, err)
	assert.Equal(t, 5, pos)
}

func TestProjectRepo_FirstPosition(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})

	pos, err := repo.FirstPosition(testUserID, p.ID, domain.StageBraindump)
	require.NoError(t, err)
	assert.Equal(t, 0, pos)

	seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageBraindump, Position: 0})
	seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageBraindump, Position: 3})
	seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageRefine, Position: -5})
	pos, err = repo.FirstPosition(testUserID, p.ID, domain.StageBraindump)
	require.NoError(t, err)
	assert.Equal(t, -1, pos)
}

func TestProjectRepo_ReorderActions_Success(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	a := seedAction(t, db, domain.Action{ProjectID: p.ID, Position: 0})
	b := seedAction(t, db, domain.Action{ProjectID: p.ID, Position: 1})

	require.NoError(t, repo.ReorderActions(testUserID, p.ID, domain.StageBraindump, []uint{b.ID, a.ID}))
	got, err := repo.GetActionsByProjectID(testUserID, p.ID)
	require.NoError(t, err)
	assert.Equal(t, b.ID, got[0].ID)
	assert.Equal(t, a.ID, got[1].ID)
}

func TestProjectRepo_ReorderActions_RejectsOtherStageOrProject(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	other := seedProject(t, db, domain.Project{Title: "Other"})
	a := seedAction(t, db, domain.Action{ProjectID: p.ID})
	refine := seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageRefine})
	foreign := seedAction(t, db, domain.Action{ProjectID: other.ID})

	err := repo.ReorderActions(testUserID, p.ID, domain.StageBraindump, []uint{a.ID, refine.ID})
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
	err = repo.ReorderActions(testUserID, p.ID, domain.StageBraindump, []uint{a.ID, foreign.ID})
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
}

func TestProjectRepo_ReorderActions_RejectsPartialOrDuplicateList(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	a := seedAction(t, db, domain.Action{ProjectID: p.ID, Position: 0})
	b := seedAction(t, db, domain.Action{ProjectID: p.ID, Position: 1})
	seedAction(t, db, domain.Action{ProjectID: p.ID, Position: 2})

	err := repo.ReorderActions(testUserID, p.ID, domain.StageBraindump, []uint{b.ID, a.ID})
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
	assert.Contains(t, err.Error(), "every action")

	err = repo.ReorderActions(testUserID, p.ID, domain.StageBraindump, []uint{a.ID, a.ID, b.ID})
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))

	got, err := repo.GetActionsByProjectID(testUserID, p.ID)
	require.NoError(t, err)
	assert.Equal(t, a.ID, got[0].ID, "rejected reorder changes nothing")
}

func TestProjectRepo_ReorderActions_WrongUser(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{UserID: otherProjectUserID})
	err := repo.ReorderActions(testUserID, p.ID, domain.StageBraindump, nil)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestProjectRepo_ReplaceWithActions_Success(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageRefine, Position: 0, Title: "existing"})
	original := seedAction(t, db, domain.Action{ProjectID: p.ID, Title: "build raised beds"})

	children := []domain.Action{
		{Title: "measure the space", Stage: domain.StageRefine},
		{Title: "buy timber", Stage: domain.StageRefine},
	}
	created, err := repo.ReplaceWithActions(testUserID, original.ID, children)
	require.NoError(t, err)
	require.Len(t, created, 2)
	assert.Equal(t, "measure the space", created[0].Title)
	assert.Equal(t, 1, created[0].Position)
	assert.Equal(t, 2, created[1].Position)
	assert.Equal(t, p.ID, created[0].ProjectID)
	assert.Equal(t, testUserID, created[0].UserID)
	assert.NotZero(t, created[0].ID)

	_, err = repo.GetActionByID(testUserID, original.ID)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	assert.Equal(t, int64(3), actionCount(t, db))
}

func TestProjectRepo_ReplaceWithActions_AtomicOnFailure(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	original := seedAction(t, db, domain.Action{ProjectID: p.ID})

	// A trigger fails the second insert; the whole thing must roll back.
	require.NoError(t, db.Exec("CREATE TRIGGER fail_second BEFORE INSERT ON actions WHEN NEW.title = 'boom' BEGIN SELECT RAISE(ABORT, 'boom'); END").Error)
	_, err := repo.ReplaceWithActions(testUserID, original.ID, []domain.Action{
		{Title: "ok", Stage: domain.StageRefine},
		{Title: "boom", Stage: domain.StageRefine},
	})
	require.Error(t, err)

	_, err = repo.GetActionByID(testUserID, original.ID)
	assert.NoError(t, err, "original must survive a failed break down")
	assert.Equal(t, int64(1), actionCount(t, db))
}

func TestProjectRepo_ReplaceWithActions_WrongUser(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{UserID: otherProjectUserID})
	a := seedAction(t, db, domain.Action{UserID: otherProjectUserID, ProjectID: p.ID})
	_, err := repo.ReplaceWithActions(testUserID, a.ID, []domain.Action{{Title: "x", Stage: domain.StageRefine}})
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	assert.Equal(t, int64(1), actionCount(t, db))
}

func TestProjectRepo_DeleteAction_WrongUser(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{UserID: otherProjectUserID})
	a := seedAction(t, db, domain.Action{UserID: otherProjectUserID, ProjectID: p.ID})
	err := repo.DeleteAction(testUserID, a.ID)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

// --- Groups ---

func seedGroup(t *testing.T, db *gorm.DB, g domain.ActionGroup) domain.ActionGroup {
	t.Helper()
	if g.UserID == 0 {
		g.UserID = testUserID
	}
	if g.Title == "" {
		g.Title = "Raised beds"
	}
	if g.Color == "" {
		g.Color = "blue"
	}
	require.NoError(t, db.Create(&g).Error)
	return g
}

func groupExists(t *testing.T, db *gorm.DB, id uint) bool {
	t.Helper()
	var n int64
	require.NoError(t, db.Unscoped().Model(&domain.ActionGroup{}).Where("id = ?", id).Count(&n).Error)
	return n > 0
}

func TestProjectRepo_CreateGroupFromAction_Success(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	original := seedAction(t, db, domain.Action{ProjectID: p.ID, Title: "build raised beds"})

	group, created, err := repo.CreateGroupFromAction(testUserID, original.ID,
		domain.ActionGroup{Title: "build raised beds", Color: "pink"},
		[]domain.Action{{Title: "measure", Stage: domain.StageRefine}, {Title: "buy timber", Stage: domain.StageRefine}})
	require.NoError(t, err)
	assert.NotZero(t, group.ID)
	assert.Equal(t, p.ID, group.ProjectID)
	assert.Equal(t, testUserID, group.UserID)
	require.Len(t, created, 2)
	for i, c := range created {
		require.NotNil(t, c.GroupID)
		assert.Equal(t, group.ID, *c.GroupID)
		assert.Equal(t, domain.StageRefine, c.Stage)
		assert.Equal(t, i, c.Position)
	}
	_, err = repo.GetActionByID(testUserID, original.ID)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestProjectRepo_CreateGroupFromAction_AtomicOnFailure(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	original := seedAction(t, db, domain.Action{ProjectID: p.ID})

	require.NoError(t, db.Exec("CREATE TRIGGER fail_boom BEFORE INSERT ON actions WHEN NEW.title = 'boom' BEGIN SELECT RAISE(ABORT, 'boom'); END").Error)
	_, _, err := repo.CreateGroupFromAction(testUserID, original.ID, domain.ActionGroup{Title: "g", Color: "blue"},
		[]domain.Action{{Title: "ok", Stage: domain.StageRefine}, {Title: "boom", Stage: domain.StageRefine}})
	require.Error(t, err)

	_, err = repo.GetActionByID(testUserID, original.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), actionCount(t, db))
	groups, _ := repo.GetGroupsByProjectID(testUserID, p.ID)
	assert.Empty(t, groups)
}

func TestProjectRepo_CreateGroupFromAction_WrongUser(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{UserID: otherProjectUserID})
	a := seedAction(t, db, domain.Action{UserID: otherProjectUserID, ProjectID: p.ID})
	_, _, err := repo.CreateGroupFromAction(testUserID, a.ID, domain.ActionGroup{Title: "g"}, []domain.Action{{Title: "x", Stage: domain.StageRefine}})
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestProjectRepo_ReplaceWithActions_KeepsGroup(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	g := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID})
	original := seedAction(t, db, domain.Action{ProjectID: p.ID, GroupID: &g.ID})

	created, err := repo.ReplaceWithActions(testUserID, original.ID,
		[]domain.Action{{Title: "a", Stage: domain.StageRefine}, {Title: "b", Stage: domain.StageRefine}})
	require.NoError(t, err)
	for _, c := range created {
		require.NotNil(t, c.GroupID)
		assert.Equal(t, g.ID, *c.GroupID)
	}
	assert.True(t, groupExists(t, db, g.ID))
}

func TestProjectRepo_SetActionGroup_IntoBetweenOut(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	g1 := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID})
	g2 := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID})
	keep := seedAction(t, db, domain.Action{ProjectID: p.ID, GroupID: &g2.ID})
	a := seedAction(t, db, domain.Action{ProjectID: p.ID, Stage: domain.StageReady, Position: 4})

	// into g1
	got, err := repo.SetActionGroup(testUserID, a.ID, &g1.ID)
	require.NoError(t, err)
	assert.Equal(t, g1.ID, *got.GroupID)

	// g1 → g2: g1 is now empty, and stays
	got, err = repo.SetActionGroup(testUserID, a.ID, &g2.ID)
	require.NoError(t, err)
	assert.Equal(t, g2.ID, *got.GroupID)
	assert.True(t, groupExists(t, db, g1.ID))

	// out of g2: keep is still there, so g2 stays
	got, err = repo.SetActionGroup(testUserID, a.ID, nil)
	require.NoError(t, err)
	assert.Nil(t, got.GroupID)
	assert.Equal(t, domain.StageReady, got.Stage)
	assert.Equal(t, 4, got.Position)
	assert.True(t, groupExists(t, db, g2.ID))

	// last one out leaves g2 empty, not removed
	_, err = repo.SetActionGroup(testUserID, keep.ID, nil)
	require.NoError(t, err)
	assert.True(t, groupExists(t, db, g2.ID))

	stored, err := repo.GetActionByID(testUserID, a.ID)
	require.NoError(t, err)
	assert.Nil(t, stored.GroupID)
}

func TestProjectRepo_SetActionGroup_RejectsForeignGroup(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	other := seedProject(t, db, domain.Project{Title: "Other"})
	foreign := seedGroup(t, db, domain.ActionGroup{ProjectID: other.ID})
	otherUsers := seedGroup(t, db, domain.ActionGroup{UserID: otherProjectUserID, ProjectID: p.ID})
	a := seedAction(t, db, domain.Action{ProjectID: p.ID})

	for _, bad := range []uint{foreign.ID, otherUsers.ID, 999} {
		_, err := repo.SetActionGroup(testUserID, a.ID, &bad)
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound), "group %d", bad)
	}
	got, err := repo.GetActionByID(testUserID, a.ID)
	require.NoError(t, err)
	assert.Nil(t, got.GroupID)
}

func TestProjectRepo_SetActionGroup_WrongUser(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{UserID: otherProjectUserID})
	a := seedAction(t, db, domain.Action{UserID: otherProjectUserID, ProjectID: p.ID})
	_, err := repo.SetActionGroup(testUserID, a.ID, nil)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
}

func TestProjectRepo_DeleteAction_KeepsTheEmptyGroup(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	g := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID})
	a := seedAction(t, db, domain.Action{ProjectID: p.ID, GroupID: &g.ID})

	require.NoError(t, repo.DeleteAction(testUserID, a.ID))
	assert.True(t, groupExists(t, db, g.ID))
	assert.True(t, errors.Is(repo.DeleteAction(testUserID, a.ID), gorm.ErrRecordNotFound))
}

func TestProjectRepo_CreateEmptyGroup(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})

	g, err := repo.CreateEmptyGroup(domain.ActionGroup{UserID: testUserID, ProjectID: p.ID, Title: "Launch", Color: "blue"})
	require.NoError(t, err)
	assert.NotZero(t, g.ID)

	groups, err := repo.GetGroupsByProjectID(testUserID, p.ID)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, "Launch", groups[0].Title)
}

func TestProjectRepo_Ungroup(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	g := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID})
	a := seedAction(t, db, domain.Action{ProjectID: p.ID, GroupID: &g.ID, Stage: domain.StageRefine, Position: 3})

	require.NoError(t, repo.Ungroup(testUserID, g.ID))
	assert.False(t, groupExists(t, db, g.ID))
	got, err := repo.GetActionByID(testUserID, a.ID)
	require.NoError(t, err)
	assert.Nil(t, got.GroupID)
	assert.Equal(t, domain.StageRefine, got.Stage)
	assert.Equal(t, 3, got.Position)
}

func TestProjectRepo_Ungroup_WrongUserChangesNothing(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{UserID: otherProjectUserID})
	g := seedGroup(t, db, domain.ActionGroup{UserID: otherProjectUserID, ProjectID: p.ID})
	a := seedAction(t, db, domain.Action{UserID: otherProjectUserID, ProjectID: p.ID, GroupID: &g.ID})

	err := repo.Ungroup(testUserID, g.ID)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	assert.True(t, groupExists(t, db, g.ID))
	got, err := repo.GetActionByID(otherProjectUserID, a.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.GroupID)
}

func TestProjectRepo_DeleteProject_RemovesGroups(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	g := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID})
	seedAction(t, db, domain.Action{ProjectID: p.ID, GroupID: &g.ID})

	require.NoError(t, repo.DeleteProject(testUserID, p.ID))
	assert.False(t, groupExists(t, db, g.ID))
}

func TestProjectRepo_CreateAction_GroupScope(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	other := seedProject(t, db, domain.Project{Title: "Other"})
	g := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID})
	foreign := seedGroup(t, db, domain.ActionGroup{ProjectID: other.ID})
	otherUsers := seedGroup(t, db, domain.ActionGroup{UserID: otherProjectUserID, ProjectID: p.ID})

	a, err := repo.CreateAction(domain.Action{UserID: testUserID, ProjectID: p.ID, Title: "x", Stage: domain.StageBraindump, GroupID: &g.ID})
	require.NoError(t, err)
	assert.Equal(t, g.ID, *a.GroupID)

	for _, bad := range []uint{foreign.ID, otherUsers.ID, 999} {
		_, err := repo.CreateAction(domain.Action{UserID: testUserID, ProjectID: p.ID, Title: "x", Stage: domain.StageBraindump, GroupID: &bad})
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound), "group %d", bad)
	}
}

func TestProjectRepo_GetGroups_OrderAndScope(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	first := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID, Title: "first"})
	second := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID, Title: "second"})
	seedGroup(t, db, domain.ActionGroup{UserID: otherProjectUserID, ProjectID: p.ID})

	groups, err := repo.GetGroupsByProjectID(testUserID, p.ID)
	require.NoError(t, err)
	require.Len(t, groups, 2)
	assert.Equal(t, first.ID, groups[0].ID)
	assert.Equal(t, second.ID, groups[1].ID)

	_, err = repo.GetGroupByID(testUserID, groups[0].ID)
	assert.NoError(t, err)
	assert.True(t, errors.Is(repo.UpdateGroup(otherProjectUserID, first.ID, map[string]interface{}{"title": "x"}), gorm.ErrRecordNotFound))
}

func TestProjectRepo_ActiveGroupColors_LeavesOutCompletedAndExcepted(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	active := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID, Color: "blue"})
	seedAction(t, db, domain.Action{ProjectID: p.ID, GroupID: &active.ID, Stage: domain.StageDone})
	seedAction(t, db, domain.Action{ProjectID: p.ID, GroupID: &active.ID, Stage: domain.StageRefine})
	completed := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID, Color: "pink"})
	seedAction(t, db, domain.Action{ProjectID: p.ID, GroupID: &completed.ID, Stage: domain.StageDone})
	other := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID, Color: "orange"})
	seedAction(t, db, domain.Action{ProjectID: p.ID, GroupID: &other.ID, Stage: domain.StageReady})
	// An empty group is active, so it holds its colour
	seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID, Color: "green"})
	// Another project's active group doesn't count
	q := seedProject(t, db, domain.Project{})
	elsewhere := seedGroup(t, db, domain.ActionGroup{ProjectID: q.ID, Color: "indigo"})
	seedAction(t, db, domain.Action{ProjectID: q.ID, GroupID: &elsewhere.ID})

	colors, err := repo.ActiveGroupColors(testUserID, p.ID, 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"blue", "orange", "green"}, colors)

	colors, err = repo.ActiveGroupColors(testUserID, p.ID, other.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"blue", "green"}, colors)

	colors, err = repo.ActiveGroupColors(otherProjectUserID, p.ID, 0)
	require.NoError(t, err)
	assert.Empty(t, colors)
}

func TestProjectRepo_GroupIsComplete(t *testing.T) {
	repo, db := newProjectRepo(t)
	p := seedProject(t, db, domain.Project{})
	g := seedGroup(t, db, domain.ActionGroup{ProjectID: p.ID})

	complete, err := repo.GroupIsComplete(testUserID, g.ID)
	require.NoError(t, err)
	assert.False(t, complete, "a group with no actions isn't complete")

	seedAction(t, db, domain.Action{ProjectID: p.ID, GroupID: &g.ID, Stage: domain.StageDone})
	open := seedAction(t, db, domain.Action{ProjectID: p.ID, GroupID: &g.ID, Stage: domain.StageReady})
	complete, err = repo.GroupIsComplete(testUserID, g.ID)
	require.NoError(t, err)
	assert.False(t, complete)

	require.NoError(t, db.Model(&open).Update("stage", domain.StageDone).Error)
	complete, err = repo.GroupIsComplete(testUserID, g.ID)
	require.NoError(t, err)
	assert.True(t, complete)

	complete, err = repo.GroupIsComplete(otherProjectUserID, g.ID)
	require.NoError(t, err)
	assert.False(t, complete, "another user sees nothing")
}
