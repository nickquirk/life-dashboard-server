package service

import (
	"errors"
	"testing"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newNoteService(repo *mocks.MockNoteRepository) Service {
	return NewServiceWithRepos(nil, nil, nil, nil, nil, nil, nil, repo)
}

func ptrBool(v bool) *bool { return &v }

// --- CreateNote ---

func TestService_CreateNote_Success(t *testing.T) {
	var captured domain.Note
	repo := &mocks.MockNoteRepository{
		CreateNoteFunc: func(n domain.Note) (domain.Note, error) {
			captured = n
			n.ID = 5
			return n, nil
		},
	}
	svc := newNoteService(repo)

	resp, err := svc.CreateNote(domain.CreateNoteRequest{
		UserID:  1,
		Title:   "My Note",
		Type:    domain.NoteTypeText,
		Content: "Hello",
		Color:   "blue",
	})
	require.NoError(t, err)

	assert.Equal(t, uint(5), resp.ID)
	assert.Equal(t, "My Note", resp.Title)
	assert.Equal(t, domain.NoteTypeText, resp.Type)
	assert.Equal(t, "Hello", resp.Content)
	assert.Equal(t, "blue", resp.Color)
	assert.Equal(t, uint(1), captured.UserID)
}

func TestService_CreateNote_ValidationError_DoesNotCallRepo(t *testing.T) {
	called := false
	repo := &mocks.MockNoteRepository{
		CreateNoteFunc: func(n domain.Note) (domain.Note, error) {
			called = true
			return n, nil
		},
	}
	svc := newNoteService(repo)

	_, err := svc.CreateNote(domain.CreateNoteRequest{UserID: 1, Type: ""})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
	assert.False(t, called)
}

func TestService_CreateNote_NonTextType_ContentNotPersisted(t *testing.T) {
	var captured domain.Note
	repo := &mocks.MockNoteRepository{
		CreateNoteFunc: func(n domain.Note) (domain.Note, error) {
			captured = n
			return n, nil
		},
	}
	svc := newNoteService(repo)

	_, err := svc.CreateNote(domain.CreateNoteRequest{
		UserID:  1,
		Type:    domain.NoteTypeChecklist,
		Content: "this should be ignored",
	})
	require.NoError(t, err)
	assert.Equal(t, "", captured.Content)
}

func TestService_CreateNote_RepoError(t *testing.T) {
	repo := &mocks.MockNoteRepository{
		CreateNoteFunc: func(n domain.Note) (domain.Note, error) {
			return domain.Note{}, errors.New("db error")
		},
	}
	svc := newNoteService(repo)

	_, err := svc.CreateNote(domain.CreateNoteRequest{UserID: 1, Type: domain.NoteTypeText})
	assert.Error(t, err)
}

// --- UpdateNote ---

func TestService_UpdateNote_NoFields_NoRepoCall(t *testing.T) {
	called := false
	repo := &mocks.MockNoteRepository{
		UpdateNoteFunc: func(userID, noteID uint, updates map[string]interface{}) error {
			called = true
			return nil
		},
	}
	svc := newNoteService(repo)

	resp, err := svc.UpdateNote(domain.UpdateNoteRequest{ID: 1, UserID: 1})
	require.NoError(t, err)
	assert.Equal(t, domain.UpdateNoteResponse{}, resp)
	assert.False(t, called)
}

func TestService_UpdateNote_TextToChecklist_TriggersConvertToList(t *testing.T) {
	convertCalled := false
	updateCalled := false
	repo := &mocks.MockNoteRepository{
		GetNoteByIDFunc: func(userID, noteID uint) (domain.Note, error) {
			return domain.Note{ID: noteID, Type: domain.NoteTypeText}, nil
		},
		ConvertNoteToListFunc: func(userID, noteID uint, newType string) error {
			convertCalled = true
			assert.Equal(t, domain.NoteTypeChecklist, newType)
			return nil
		},
		UpdateNoteFunc: func(userID, noteID uint, updates map[string]interface{}) error {
			updateCalled = true
			_, hasType := updates["type"]
			assert.False(t, hasType, "type should not be in regular updates map")
			_, hasContent := updates["content"]
			assert.False(t, hasContent, "content should not be in regular updates map")
			return nil
		},
	}
	svc := newNoteService(repo)

	typ := domain.NoteTypeChecklist
	_, err := svc.UpdateNote(domain.UpdateNoteRequest{ID: 1, UserID: 1, Type: &typ, Title: ptrStr("New")})
	require.NoError(t, err)
	assert.True(t, convertCalled)
	assert.True(t, updateCalled)
}

func TestService_UpdateNote_ChecklistToText_TriggersConvertToText(t *testing.T) {
	convertCalled := false
	repo := &mocks.MockNoteRepository{
		GetNoteByIDFunc: func(userID, noteID uint) (domain.Note, error) {
			return domain.Note{ID: noteID, Type: domain.NoteTypeChecklist}, nil
		},
		ConvertNoteToTextFunc: func(userID, noteID uint) error {
			convertCalled = true
			return nil
		},
		UpdateNoteFunc: func(userID, noteID uint, updates map[string]interface{}) error {
			_, hasType := updates["type"]
			assert.False(t, hasType, "type should not be in updates map after conversion")
			return nil
		},
	}
	svc := newNoteService(repo)

	typ := domain.NoteTypeText
	_, err := svc.UpdateNote(domain.UpdateNoteRequest{ID: 1, UserID: 1, Type: &typ, Title: ptrStr("T")})
	require.NoError(t, err)
	assert.True(t, convertCalled)
}

func TestService_UpdateNote_ChecklistToBullet_PlainTypeUpdate(t *testing.T) {
	convertToListCalled := false
	convertToTextCalled := false
	var captured map[string]interface{}
	repo := &mocks.MockNoteRepository{
		GetNoteByIDFunc: func(userID, noteID uint) (domain.Note, error) {
			return domain.Note{ID: noteID, Type: domain.NoteTypeChecklist}, nil
		},
		ConvertNoteToListFunc: func(userID, noteID uint, newType string) error {
			convertToListCalled = true
			return nil
		},
		ConvertNoteToTextFunc: func(userID, noteID uint) error {
			convertToTextCalled = true
			return nil
		},
		UpdateNoteFunc: func(userID, noteID uint, updates map[string]interface{}) error {
			captured = updates
			return nil
		},
	}
	svc := newNoteService(repo)

	typ := domain.NoteTypeBullet
	_, err := svc.UpdateNote(domain.UpdateNoteRequest{ID: 1, UserID: 1, Type: &typ})
	require.NoError(t, err)
	assert.False(t, convertToListCalled)
	assert.False(t, convertToTextCalled)
	assert.Equal(t, domain.NoteTypeBullet, captured["type"])
}

func TestService_UpdateNote_PinAndArchive(t *testing.T) {
	var captured map[string]interface{}
	repo := &mocks.MockNoteRepository{
		UpdateNoteFunc: func(userID, noteID uint, updates map[string]interface{}) error {
			captured = updates
			return nil
		},
	}
	svc := newNoteService(repo)

	_, err := svc.UpdateNote(domain.UpdateNoteRequest{
		ID: 1, UserID: 1,
		IsPinned:   ptrBool(true),
		IsArchived: ptrBool(false),
	})
	require.NoError(t, err)
	assert.Equal(t, true, captured["is_pinned"])
	assert.Equal(t, false, captured["is_archived"])
}

// --- ReorderNoteItems ---

func TestService_ReorderNoteItems_PassesIDsThrough(t *testing.T) {
	var capturedIDs []uint
	repo := &mocks.MockNoteRepository{
		ReorderNoteItemsFunc: func(userID, noteID uint, orderedItemIDs []uint) error {
			capturedIDs = orderedItemIDs
			return nil
		},
	}
	svc := newNoteService(repo)

	resp, err := svc.ReorderNoteItems(domain.ReorderNoteItemsRequest{
		UserID: 1, NoteID: 2, ItemIDs: []uint{3, 1, 2},
	})
	require.NoError(t, err)
	assert.Equal(t, uint(2), resp.NoteID)
	assert.Equal(t, []uint{3, 1, 2}, capturedIDs)
}

// --- UpdateNoteItem ---

func TestService_UpdateNoteItem_BuildsCorrectUpdatesMap(t *testing.T) {
	var captured map[string]interface{}
	repo := &mocks.MockNoteRepository{
		UpdateNoteItemFunc: func(userID, noteID, itemID uint, updates map[string]interface{}) error {
			captured = updates
			return nil
		},
	}
	svc := newNoteService(repo)

	pos := 3
	_, err := svc.UpdateNoteItem(domain.UpdateNoteItemRequest{
		ID: 1, UserID: 1, NoteID: 2,
		Content:     ptrStr("New content"),
		IsCompleted: ptrBool(true),
		Position:    &pos,
	})
	require.NoError(t, err)
	assert.Equal(t, "New content", captured["content"])
	assert.Equal(t, true, captured["is_completed"])
	assert.Equal(t, 3, captured["position"])
}

func TestService_UpdateNoteItem_NoFields_NoRepoCall(t *testing.T) {
	called := false
	repo := &mocks.MockNoteRepository{
		UpdateNoteItemFunc: func(userID, noteID, itemID uint, updates map[string]interface{}) error {
			called = true
			return nil
		},
	}
	svc := newNoteService(repo)

	resp, err := svc.UpdateNoteItem(domain.UpdateNoteItemRequest{ID: 1, UserID: 1, NoteID: 2})
	require.NoError(t, err)
	assert.Equal(t, domain.UpdateNoteItemResponse{}, resp)
	assert.False(t, called)
}

// --- DeleteNote ---

func TestService_DeleteNote_NotFound(t *testing.T) {
	repo := &mocks.MockNoteRepository{
		DeleteNoteFunc: func(userID, noteID uint) error {
			return gorm.ErrRecordNotFound
		},
	}
	svc := newNoteService(repo)

	_, err := svc.DeleteNote(domain.DeleteNoteRequest{ID: 99, UserID: 1})
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
