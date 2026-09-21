package repository

import (
	"testing"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newUserRepo(t *testing.T) (*GormUserRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Session{},
		&domain.TaskList{},
		&domain.Task{},
		&domain.Zone{},
		&domain.UserSettings{},
		&domain.Feedback{},
		&domain.Scratchpad{},
		&domain.Routine{},
		&domain.RoutineInstance{},
		&domain.Note{},
		&domain.NoteItem{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return &GormUserRepository{Db: db}, db
}

// TestUserRepo_DeleteUserAndDataRemovesAllUserRows seeds one row per table that
// hangs off a user (directly or transitively) and asserts DeleteUserAndData
// leaves none of them behind, including soft-deletable rows that a plain
// scoped query would otherwise hide.
func TestUserRepo_DeleteUserAndDataRemovesAllUserRows(t *testing.T) {
	repo, db := newUserRepo(t)

	user := domain.User{Email: "delete-me@example.com"}
	require.NoError(t, db.Create(&user).Error)

	otherUser := domain.User{Email: "keep-me@example.com"}
	require.NoError(t, db.Create(&otherUser).Error)

	require.NoError(t, db.Create(&domain.Session{
		UserID:          user.ID,
		AppRefreshToken: "hashed-token",
		ExpiresAt:       time.Now().Add(time.Hour),
	}).Error)

	taskList := domain.TaskList{ID: "tl1", UserID: user.ID, Title: "List"}
	require.NoError(t, db.Create(&taskList).Error)

	parentTask := domain.Task{ID: "t1", TaskListID: taskList.ID, Title: "Parent"}
	require.NoError(t, db.Create(&parentTask).Error)
	childTask := domain.Task{ID: "t2", TaskListID: taskList.ID, Parent: &parentTask.ID, Title: "Child"}
	require.NoError(t, db.Create(&childTask).Error)

	require.NoError(t, db.Create(&domain.Zone{UserID: user.ID, Label: "Zone"}).Error)

	require.NoError(t, db.Create(&domain.UserSettings{UserID: user.ID, Data: domain.DefaultSettings()}).Error)

	require.NoError(t, db.Create(&domain.Feedback{UserID: user.ID, Type: "bug", Message: "test"}).Error)

	require.NoError(t, db.Create(&domain.Scratchpad{UserId: user.ID, Date: "2024-01-01", Content: "note"}).Error)

	routine := domain.Routine{UserID: user.ID, Title: "Routine", DurationMins: 10}
	require.NoError(t, db.Create(&routine).Error)
	require.NoError(t, db.Create(&domain.RoutineInstance{
		UserID:    user.ID,
		RoutineID: routine.ID,
		Date:      time.Now(),
		Status:    "needsAction",
	}).Error)

	note := domain.Note{UserID: user.ID, Title: "Note", Type: domain.NoteTypeChecklist}
	require.NoError(t, db.Create(&note).Error)
	require.NoError(t, db.Create(&domain.NoteItem{NoteID: note.ID, Content: "item"}).Error)

	// A row belonging to another user that must survive.
	otherZone := domain.Zone{UserID: otherUser.ID, Label: "Keep"}
	require.NoError(t, db.Create(&otherZone).Error)

	require.NoError(t, repo.DeleteUserAndData(user.ID))

	assertGone(t, db, "user", &domain.User{}, "id = ?", user.ID)
	assertGone(t, db, "session", &domain.Session{}, "user_id = ?", user.ID)
	assertGone(t, db, "task_list", &domain.TaskList{}, "user_id = ?", user.ID)
	assertGone(t, db, "task", &domain.Task{}, "task_list_id = ?", taskList.ID)
	assertGone(t, db, "zone", &domain.Zone{}, "user_id = ?", user.ID)
	assertGone(t, db, "user_settings", &domain.UserSettings{}, "user_id = ?", user.ID)
	assertGone(t, db, "feedback", &domain.Feedback{}, "user_id = ?", user.ID)
	assertGone(t, db, "scratchpad", &domain.Scratchpad{}, "user_id = ?", user.ID)
	assertGone(t, db, "routine", &domain.Routine{}, "user_id = ?", user.ID)
	assertGone(t, db, "routine_instance", &domain.RoutineInstance{}, "user_id = ?", user.ID)
	assertGone(t, db, "note", &domain.Note{}, "user_id = ?", user.ID)
	assertGone(t, db, "note_item", &domain.NoteItem{}, "note_id = ?", note.ID)

	// The other user's data must be untouched.
	var otherZoneCount int64
	require.NoError(t, db.Model(&domain.Zone{}).Where("user_id = ?", otherUser.ID).Count(&otherZoneCount).Error)
	require.Equal(t, int64(1), otherZoneCount, "unrelated user's rows must not be deleted")
}

func assertGone(t *testing.T, db *gorm.DB, label string, model interface{}, where string, args ...interface{}) {
	t.Helper()
	var count int64
	require.NoError(t, db.Unscoped().Model(model).Where(where, args...).Count(&count).Error)
	require.Equal(t, int64(0), count, "expected no %s rows to remain after user deletion", label)
}
