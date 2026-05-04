package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- CreateNoteRequest.Validate ---

func TestCreateNoteRequest_Valid_Text(t *testing.T) {
	req := CreateNoteRequest{Title: "My Note", Type: NoteTypeText, Content: "hello"}
	require.NoError(t, req.Validate())
}

func TestCreateNoteRequest_Valid_Checklist(t *testing.T) {
	req := CreateNoteRequest{Type: NoteTypeChecklist}
	require.NoError(t, req.Validate())
}

func TestCreateNoteRequest_Valid_Bullet(t *testing.T) {
	req := CreateNoteRequest{Type: NoteTypeBullet}
	require.NoError(t, req.Validate())
}

func TestCreateNoteRequest_TrimsTitle(t *testing.T) {
	req := CreateNoteRequest{Title: "  Note  ", Type: NoteTypeText}
	require.NoError(t, req.Validate())
	assert.Equal(t, "Note", req.Title)
}

func TestCreateNoteRequest_EmptyTitleAllowed(t *testing.T) {
	req := CreateNoteRequest{Type: NoteTypeText}
	require.NoError(t, req.Validate())
}

func TestCreateNoteRequest_TitleTooLong(t *testing.T) {
	req := CreateNoteRequest{Title: strings.Repeat("a", maxNoteTitleLen+1), Type: NoteTypeText}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
	assert.Contains(t, err.Error(), "title")
}

func TestCreateNoteRequest_InvalidType(t *testing.T) {
	req := CreateNoteRequest{Type: "invalid"}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
	assert.Contains(t, err.Error(), "type")
}

func TestCreateNoteRequest_EmptyTypeInvalid(t *testing.T) {
	req := CreateNoteRequest{Type: ""}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

func TestCreateNoteRequest_ContentTooLong_TextType(t *testing.T) {
	req := CreateNoteRequest{Type: NoteTypeText, Content: strings.Repeat("x", maxNoteContentLen+1)}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
	assert.Contains(t, err.Error(), "content")
}

func TestCreateNoteRequest_ContentTooLong_NonTextTypeAllowed(t *testing.T) {
	// Non-text types don't enforce content cap; the service strips it anyway.
	req := CreateNoteRequest{Type: NoteTypeChecklist, Content: strings.Repeat("x", maxNoteContentLen+1)}
	require.NoError(t, req.Validate())
}

func TestCreateNoteRequest_ColorTooLong(t *testing.T) {
	req := CreateNoteRequest{Type: NoteTypeText, Color: strings.Repeat("c", maxNoteColorLen+1)}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
	assert.Contains(t, err.Error(), "color")
}

func TestCreateNoteRequest_ColorMaxLength_OK(t *testing.T) {
	req := CreateNoteRequest{Type: NoteTypeText, Color: strings.Repeat("c", maxNoteColorLen)}
	require.NoError(t, req.Validate())
}

// --- UpdateNoteRequest.Validate ---

func TestUpdateNoteRequest_NoFields_OK(t *testing.T) {
	req := UpdateNoteRequest{}
	require.NoError(t, req.Validate())
}

func TestUpdateNoteRequest_TrimsTitle(t *testing.T) {
	title := "  Updated  "
	req := UpdateNoteRequest{Title: &title}
	require.NoError(t, req.Validate())
	assert.Equal(t, "Updated", *req.Title)
}

func TestUpdateNoteRequest_TitleTooLong(t *testing.T) {
	title := strings.Repeat("a", maxNoteTitleLen+1)
	req := UpdateNoteRequest{Title: &title}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

func TestUpdateNoteRequest_InvalidType(t *testing.T) {
	typ := "not-a-type"
	req := UpdateNoteRequest{Type: &typ}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

func TestUpdateNoteRequest_ValidTypes(t *testing.T) {
	for _, typ := range []string{NoteTypeText, NoteTypeChecklist, NoteTypeBullet} {
		tp := typ
		req := UpdateNoteRequest{Type: &tp}
		assert.NoError(t, req.Validate(), "type=%q", typ)
	}
}

func TestUpdateNoteRequest_ContentTooLong(t *testing.T) {
	c := strings.Repeat("x", maxNoteContentLen+1)
	req := UpdateNoteRequest{Content: &c}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

func TestUpdateNoteRequest_ColorTooLong(t *testing.T) {
	c := strings.Repeat("c", maxNoteColorLen+1)
	req := UpdateNoteRequest{Color: &c}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

// --- CreateNoteItemRequest.Validate ---

func TestCreateNoteItemRequest_Valid(t *testing.T) {
	req := CreateNoteItemRequest{Content: "Buy milk", Position: 0}
	require.NoError(t, req.Validate())
}

func TestCreateNoteItemRequest_EmptyContentInvalid(t *testing.T) {
	req := CreateNoteItemRequest{Content: "   ", Position: 0}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
	assert.Contains(t, err.Error(), "content")
}

func TestCreateNoteItemRequest_ContentTooLong(t *testing.T) {
	req := CreateNoteItemRequest{Content: strings.Repeat("x", maxNoteItemContentLen+1), Position: 0}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

func TestCreateNoteItemRequest_NegativePosition(t *testing.T) {
	req := CreateNoteItemRequest{Content: "hello", Position: -1}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
	assert.Contains(t, err.Error(), "position")
}

func TestCreateNoteItemRequest_ZeroPositionOK(t *testing.T) {
	req := CreateNoteItemRequest{Content: "hello", Position: 0}
	require.NoError(t, req.Validate())
}

// --- UpdateNoteItemRequest.Validate ---

func TestUpdateNoteItemRequest_NoFields_OK(t *testing.T) {
	req := UpdateNoteItemRequest{}
	require.NoError(t, req.Validate())
}

func TestUpdateNoteItemRequest_ContentTooLong(t *testing.T) {
	c := strings.Repeat("x", maxNoteItemContentLen+1)
	req := UpdateNoteItemRequest{Content: &c}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

func TestUpdateNoteItemRequest_NegativePosition(t *testing.T) {
	p := -1
	req := UpdateNoteItemRequest{Position: &p}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

func TestUpdateNoteItemRequest_ZeroPositionOK(t *testing.T) {
	p := 0
	req := UpdateNoteItemRequest{Position: &p}
	require.NoError(t, req.Validate())
}
