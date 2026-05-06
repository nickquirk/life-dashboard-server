package repository

import (
	"testing"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.TaskList{}, &domain.Task{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newTaskRepo(t *testing.T) (*GormTaskRepository, *gorm.DB) {
	t.Helper()
	db := newTaskTestDB(t)
	return &GormTaskRepository{Db: db}, db
}

func seedTaskList(t *testing.T, db *gorm.DB, list domain.TaskList) domain.TaskList {
	t.Helper()
	if list.UserID == 0 {
		list.UserID = testUserID
	}
	if list.ID == "" {
		list.ID = "list-" + list.Title
	}
	require.NoError(t, db.Create(&list).Error)
	return list
}

func seedTask(t *testing.T, db *gorm.DB, task domain.Task) domain.Task {
	t.Helper()
	require.NoError(t, db.Create(&task).Error)
	return task
}

// ReorderSubtasks: positions update for the named children.
func TestTaskRepo_ReorderSubtasks_Success(t *testing.T) {
	repo, db := newTaskRepo(t)
	list := seedTaskList(t, db, domain.TaskList{Title: "L1"})
	parent := seedTask(t, db, domain.Task{ID: "p1", TaskListID: list.ID, Title: "Parent"})
	parentID := parent.ID
	a := seedTask(t, db, domain.Task{ID: "a", TaskListID: list.ID, Parent: &parentID, Title: "A", Position: 0})
	b := seedTask(t, db, domain.Task{ID: "b", TaskListID: list.ID, Parent: &parentID, Title: "B", Position: 1})

	require.NoError(t, repo.ReorderSubtasks(testUserID, parent.ID, []string{b.ID, a.ID}))

	var got []domain.Task
	require.NoError(t, db.Where("parent = ?", parent.ID).Order("position ASC").Find(&got).Error)
	require.Len(t, got, 2)
	assert.Equal(t, b.ID, got[0].ID)
	assert.Equal(t, a.ID, got[1].ID)
}

// ReorderSubtasks: rejects ID that isn't a child of the parent.
func TestTaskRepo_ReorderSubtasks_WrongChild(t *testing.T) {
	repo, db := newTaskRepo(t)
	list := seedTaskList(t, db, domain.TaskList{Title: "L1"})
	parent := seedTask(t, db, domain.Task{ID: "p1", TaskListID: list.ID, Title: "Parent"})
	parentID := parent.ID
	a := seedTask(t, db, domain.Task{ID: "a", TaskListID: list.ID, Parent: &parentID, Title: "A"})
	// Sibling-of-different-parent
	stranger := seedTask(t, db, domain.Task{ID: "x", TaskListID: list.ID, Title: "Top-level"})

	err := repo.ReorderSubtasks(testUserID, parent.ID, []string{a.ID, stranger.ID})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

// ReorderSubtasks: rejects when the parent's task list isn't owned by the user.
func TestTaskRepo_ReorderSubtasks_WrongUser(t *testing.T) {
	repo, db := newTaskRepo(t)
	list := seedTaskList(t, db, domain.TaskList{UserID: 99, Title: "Other"})
	parent := seedTask(t, db, domain.Task{ID: "p1", TaskListID: list.ID, Title: "Parent"})
	parentID := parent.ID
	a := seedTask(t, db, domain.Task{ID: "a", TaskListID: list.ID, Parent: &parentID, Title: "A"})

	err := repo.ReorderSubtasks(testUserID, parent.ID, []string{a.ID})
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// GetTasks orders by position then created_at so subtasks render stably.
func TestTaskRepo_GetTasks_OrdersByPosition(t *testing.T) {
	repo, db := newTaskRepo(t)
	list := seedTaskList(t, db, domain.TaskList{Title: "L1"})
	parent := seedTask(t, db, domain.Task{ID: "p1", TaskListID: list.ID, Title: "Parent"})
	parentID := parent.ID
	seedTask(t, db, domain.Task{ID: "second", TaskListID: list.ID, Parent: &parentID, Title: "Second", Position: 1})
	seedTask(t, db, domain.Task{ID: "first", TaskListID: list.ID, Parent: &parentID, Title: "First", Position: 0})

	got, err := repo.GetTasks(list.ID)
	require.NoError(t, err)

	// Filter to subtasks for assertion.
	var children []domain.Task
	for _, task := range got {
		if task.Parent != nil && *task.Parent == parent.ID {
			children = append(children, task)
		}
	}
	require.Len(t, children, 2)
	assert.Equal(t, "first", children[0].ID)
	assert.Equal(t, "second", children[1].ID)
}
