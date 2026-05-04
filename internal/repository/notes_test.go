package repository

import (
	"testing"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newNoteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.Note{}, &domain.NoteItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newNoteRepo(t *testing.T) (*GormNoteRepository, *gorm.DB) {
	t.Helper()
	db := newNoteTestDB(t)
	return &GormNoteRepository{Db: db}, db
}

func seedNote(t *testing.T, db *gorm.DB, n domain.Note) domain.Note {
	t.Helper()
	if n.UserID == 0 {
		n.UserID = testUserID
	}
	if n.Type == "" {
		n.Type = domain.NoteTypeText
	}
	require.NoError(t, db.Create(&n).Error)
	return n
}

func seedNoteItem(t *testing.T, db *gorm.DB, item domain.NoteItem) domain.NoteItem {
	t.Helper()
	require.NoError(t, db.Create(&item).Error)
	return item
}

// Create + GetByUserID returns note with items preloaded ordered by position.
func TestNoteRepo_CreateAndGet_WithItemsPreloaded(t *testing.T) {
	repo, db := newNoteRepo(t)
	n := seedNote(t, db, domain.Note{Type: domain.NoteTypeChecklist, Title: "Shopping"})
	seedNoteItem(t, db, domain.NoteItem{NoteID: n.ID, Content: "Bread", Position: 1})
	seedNoteItem(t, db, domain.NoteItem{NoteID: n.ID, Content: "Apples", Position: 0})

	notes, err := repo.GetNotesByUserID(testUserID)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	require.Len(t, notes[0].Items, 2)
	assert.Equal(t, "Apples", notes[0].Items[0].Content)
	assert.Equal(t, "Bread", notes[0].Items[1].Content)
}

// GetByUserID excludes other users' notes.
func TestNoteRepo_GetByUserID_MultiUserIsolation(t *testing.T) {
	const otherUser uint = 99
	repo, db := newNoteRepo(t)
	seedNote(t, db, domain.Note{UserID: testUserID, Title: "Mine"})
	seedNote(t, db, domain.Note{UserID: otherUser, Title: "Theirs"})

	notes, err := repo.GetNotesByUserID(testUserID)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "Mine", notes[0].Title)
}

// GetByUserID excludes archived notes.
func TestNoteRepo_GetByUserID_ExcludesArchived(t *testing.T) {
	repo, db := newNoteRepo(t)
	seedNote(t, db, domain.Note{Title: "Active"})
	seedNote(t, db, domain.Note{Title: "Archived", IsArchived: true})

	notes, err := repo.GetNotesByUserID(testUserID)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "Active", notes[0].Title)
}

// GetByUserID returns pinned notes first.
func TestNoteRepo_GetByUserID_PinnedFirst(t *testing.T) {
	repo, db := newNoteRepo(t)
	seedNote(t, db, domain.Note{Title: "Normal"})
	seedNote(t, db, domain.Note{Title: "Pinned", IsPinned: true})

	notes, err := repo.GetNotesByUserID(testUserID)
	require.NoError(t, err)
	require.Len(t, notes, 2)
	assert.Equal(t, "Pinned", notes[0].Title)
}

// UpdateNote returns ErrRecordNotFound when note doesn't belong to user.
func TestNoteRepo_UpdateNote_WrongUser(t *testing.T) {
	repo, db := newNoteRepo(t)
	n := seedNote(t, db, domain.Note{UserID: 2, Title: "Other"})

	err := repo.UpdateNote(testUserID, n.ID, map[string]interface{}{"title": "Hacked"})
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// DeleteNote returns ErrRecordNotFound for wrong user.
func TestNoteRepo_DeleteNote_WrongUser(t *testing.T) {
	repo, db := newNoteRepo(t)
	n := seedNote(t, db, domain.Note{UserID: 2, Title: "Other"})

	err := repo.DeleteNote(testUserID, n.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// ConvertNoteToList: text -> checklist creates items, blanks content, sets type.
func TestNoteRepo_ConvertNoteToList_FromText(t *testing.T) {
	repo, db := newNoteRepo(t)
	n := seedNote(t, db, domain.Note{Title: "Todo", Type: domain.NoteTypeText, Content: "Buy milk\nBuy eggs\n\nBuy bread"})

	err := repo.ConvertNoteToList(testUserID, n.ID, domain.NoteTypeChecklist)
	require.NoError(t, err)

	var updated domain.Note
	require.NoError(t, db.Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("position ASC")
	}).First(&updated, n.ID).Error)

	assert.Equal(t, domain.NoteTypeChecklist, updated.Type)
	assert.Equal(t, "", updated.Content)
	require.Len(t, updated.Items, 3)
	assert.Equal(t, "Buy milk", updated.Items[0].Content)
	assert.Equal(t, "Buy eggs", updated.Items[1].Content)
	assert.Equal(t, "Buy bread", updated.Items[2].Content)
}

// ConvertNoteToList: empty content yields zero items.
func TestNoteRepo_ConvertNoteToList_EmptyContent_NoItems(t *testing.T) {
	repo, db := newNoteRepo(t)
	n := seedNote(t, db, domain.Note{Type: domain.NoteTypeText, Content: ""})

	err := repo.ConvertNoteToList(testUserID, n.ID, domain.NoteTypeChecklist)
	require.NoError(t, err)

	var items []domain.NoteItem
	require.NoError(t, db.Where("note_id = ?", n.ID).Find(&items).Error)
	assert.Empty(t, items)
}

// ConvertNoteToList: rejects non-text note.
func TestNoteRepo_ConvertNoteToList_RejectsNonText(t *testing.T) {
	repo, db := newNoteRepo(t)
	n := seedNote(t, db, domain.Note{Type: domain.NoteTypeChecklist})

	err := repo.ConvertNoteToList(testUserID, n.ID, domain.NoteTypeBullet)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

// ConvertNoteToText: round-trips items into \n-joined content and deletes items.
func TestNoteRepo_ConvertNoteToText_RoundTrip(t *testing.T) {
	repo, db := newNoteRepo(t)
	n := seedNote(t, db, domain.Note{Type: domain.NoteTypeChecklist, Title: "Checklist"})
	seedNoteItem(t, db, domain.NoteItem{NoteID: n.ID, Content: "Alpha", Position: 0})
	seedNoteItem(t, db, domain.NoteItem{NoteID: n.ID, Content: "Beta", Position: 1})

	err := repo.ConvertNoteToText(testUserID, n.ID)
	require.NoError(t, err)

	var updated domain.Note
	require.NoError(t, db.First(&updated, n.ID).Error)
	assert.Equal(t, domain.NoteTypeText, updated.Type)
	assert.Equal(t, "Alpha\nBeta", updated.Content)

	var items []domain.NoteItem
	require.NoError(t, db.Unscoped().Where("note_id = ?", n.ID).Find(&items).Error)
	assert.Empty(t, items)
}

// ReorderNoteItems: positions updated correctly.
func TestNoteRepo_ReorderNoteItems_Success(t *testing.T) {
	repo, db := newNoteRepo(t)
	n := seedNote(t, db, domain.Note{Type: domain.NoteTypeChecklist})
	item1 := seedNoteItem(t, db, domain.NoteItem{NoteID: n.ID, Content: "First", Position: 0})
	item2 := seedNoteItem(t, db, domain.NoteItem{NoteID: n.ID, Content: "Second", Position: 1})

	err := repo.ReorderNoteItems(testUserID, n.ID, []uint{item2.ID, item1.ID})
	require.NoError(t, err)

	var reordered []domain.NoteItem
	require.NoError(t, db.Where("note_id = ?", n.ID).Order("position ASC").Find(&reordered).Error)
	assert.Equal(t, item2.ID, reordered[0].ID)
	assert.Equal(t, item1.ID, reordered[1].ID)
}

// ReorderNoteItems: rejects item ID that doesn't belong to the note.
func TestNoteRepo_ReorderNoteItems_WrongItem(t *testing.T) {
	repo, db := newNoteRepo(t)
	n := seedNote(t, db, domain.Note{Type: domain.NoteTypeChecklist})
	item := seedNoteItem(t, db, domain.NoteItem{NoteID: n.ID, Content: "Item", Position: 0})

	err := repo.ReorderNoteItems(testUserID, n.ID, []uint{item.ID, 9999})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

// ReorderNoteItems: rejects when noteID isn't owned by user.
func TestNoteRepo_ReorderNoteItems_WrongUser(t *testing.T) {
	repo, db := newNoteRepo(t)
	n := seedNote(t, db, domain.Note{UserID: 2, Type: domain.NoteTypeChecklist})
	item := seedNoteItem(t, db, domain.NoteItem{NoteID: n.ID, Content: "Item", Position: 0})

	err := repo.ReorderNoteItems(testUserID, n.ID, []uint{item.ID})
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// UpdateNoteItem: cross-user attempt returns ErrRecordNotFound.
func TestNoteRepo_UpdateNoteItem_CrossUser(t *testing.T) {
	repo, db := newNoteRepo(t)
	n := seedNote(t, db, domain.Note{UserID: 2, Type: domain.NoteTypeChecklist})
	item := seedNoteItem(t, db, domain.NoteItem{NoteID: n.ID, Content: "Item", Position: 0})

	err := repo.UpdateNoteItem(testUserID, n.ID, item.ID, map[string]interface{}{"content": "Hacked"})
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// DeleteNoteItem: cross-user attempt returns ErrRecordNotFound.
func TestNoteRepo_DeleteNoteItem_CrossUser(t *testing.T) {
	repo, db := newNoteRepo(t)
	n := seedNote(t, db, domain.Note{UserID: 2, Type: domain.NoteTypeChecklist})
	item := seedNoteItem(t, db, domain.NoteItem{NoteID: n.ID, Content: "Item", Position: 0})

	err := repo.DeleteNoteItem(testUserID, n.ID, item.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
