package db

import (
	"testing"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// legacyAction is the actions table from before slice 5, which still had date.
type legacyAction struct {
	domain.Action
	Date *time.Time `gorm:"type:datetime"`
}

func (legacyAction) TableName() string { return "actions" }

// legacyDB is a database whose actions table still has date and its index.
func legacyDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Project{}, &legacyAction{}, &domain.ActionGroup{}))
	require.NoError(t, db.Exec("CREATE INDEX idx_actions_user_date ON actions(user_id, date)").Error)
	return db
}

func seedLegacy(t *testing.T, db *gorm.DB, rows []legacyAction) {
	t.Helper()
	for i := range rows {
		rows[i].UserID = 1
		require.NoError(t, db.Create(&rows[i]).Error)
	}
}

func TestBackfillActionCompletedAt(t *testing.T) {
	db := legacyDB(t)

	day := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	kept := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	rows := []legacyAction{
		{Action: domain.Action{Title: "done dated", Stage: domain.StageDone, ProjectID: 1}, Date: &day},
		{Action: domain.Action{Title: "done undated", Stage: domain.StageDone, ProjectID: 1}},
		{Action: domain.Action{Title: "done filled", Stage: domain.StageDone, ProjectID: 1, CompletedAt: &kept}, Date: &day},
		{Action: domain.Action{Title: "ready", Stage: domain.StageReady, ProjectID: 1}},
	}
	seedLegacy(t, db, rows)

	require.NoError(t, BackfillActionCompletedAt(db))

	get := func(id uint) domain.Action {
		var a domain.Action
		require.NoError(t, db.First(&a, id).Error)
		return a
	}
	dated := get(rows[0].ID)
	require.NotNil(t, dated.CompletedAt)
	assert.True(t, dated.CompletedAt.Equal(day))
	assert.NotNil(t, get(rows[1].ID).CompletedAt, "undated falls back to updated_at")
	assert.True(t, get(rows[2].ID).CompletedAt.Equal(kept), "already filled is left alone")
	assert.Nil(t, get(rows[3].ID).CompletedAt)
}

func TestInitMigration_RetiresScheduling(t *testing.T) {
	db := legacyDB(t)
	at := func(d int) *time.Time { t := time.Date(2026, 11, d, 9, 0, 0, 0, time.UTC); return &t }
	done := at(1)
	rows := []legacyAction{
		{Action: domain.Action{Title: "p1 ready", ProjectID: 1, Stage: domain.StageReady, Position: 4}},
		{Action: domain.Action{Title: "p1 later", ProjectID: 1, Stage: "scheduled"}, Date: at(20)},
		{Action: domain.Action{Title: "p1 sooner", ProjectID: 1, Stage: "scheduled"}, Date: at(18)},
		{Action: domain.Action{Title: "p1 done", ProjectID: 1, Stage: domain.StageDone, CompletedAt: done}, Date: at(2)},
		{Action: domain.Action{Title: "p2 only", ProjectID: 2, Stage: "scheduled"}, Date: at(19)},
	}
	seedLegacy(t, db, rows)

	require.NoError(t, InitMigration(db))

	get := func(i int) domain.Action {
		var a domain.Action
		require.NoError(t, db.First(&a, rows[i].ID).Error)
		return a
	}
	assert.Equal(t, 4, get(0).Position)
	for i, want := range map[int]int{2: 5, 1: 6, 4: 0} {
		assert.Equal(t, domain.StageReady, get(i).Stage, rows[i].Title)
		assert.Equal(t, want, get(i).Position, rows[i].Title)
	}
	d := get(3)
	assert.Equal(t, domain.StageDone, d.Stage)
	assert.True(t, d.CompletedAt.Equal(*done))

	m := db.Migrator()
	assert.False(t, m.HasColumn(&domain.Action{}, "date"))
	assert.False(t, m.HasIndex(&domain.Action{}, "idx_actions_user_date"))

	require.NoError(t, InitMigration(db), "second run is a no-op")
	assert.Equal(t, 6, get(1).Position)
}

func TestInitMigration_FreshDatabase(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, InitMigration(db))
	assert.False(t, db.Migrator().HasColumn(&domain.Action{}, "date"))
}

func TestInitMigration_ReturnsError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	assert.ErrorContains(t, InitMigration(db), "migrate users")
}

// A DB foreign key from actions to work_sessions made MySQL migrate actions
// before work_sessions existed, so work_sessions was never created.
func TestInitMigration_NoWorkSessionForeignKeyOnActions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, InitMigration(db))

	assert.True(t, db.Migrator().HasTable(&domain.WorkSession{}))
	assert.True(t, db.Migrator().HasColumn(&domain.Action{}, "work_session_id"))
	var ddl string
	require.NoError(t, db.Raw(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'actions'`).Scan(&ddl).Error)
	assert.NotContains(t, ddl, "work_sessions")
	require.NoError(t, InitMigration(db), "second run is a no-op")
}
