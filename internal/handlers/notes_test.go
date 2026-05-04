package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// withChi adds a single chi URL param to an existing request context.
func withChi(r *http.Request, key, val string) *http.Request {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		rctx = chi.NewRouteContext()
		ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
		r = r.WithContext(ctx)
	}
	rctx.URLParams.Add(key, val)
	return r
}

// --- createNote ---

func TestCreateNote_Success(t *testing.T) {
	svc := &mocks.MockService{
		CreateNoteFunc: func(req domain.CreateNoteRequest) (domain.CreateNoteResponse, error) {
			return domain.CreateNoteResponse{ID: 3, Title: req.Title, Type: req.Type}, nil
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.CreateNoteRequest{Title: "My Note", Type: domain.NoteTypeText})
	r := httptest.NewRequest(http.MethodPost, "/api/notes", bytes.NewReader(body))
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.createNote(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"id":3`)
}

func TestCreateNote_NoUser(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPost, "/api/notes", nil)
	rr := httptest.NewRecorder()

	h.createNote(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestCreateNote_BadBody(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPost, "/api/notes", bytes.NewReader([]byte("not json")))
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.createNote(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateNote_ValidationError_Returns400(t *testing.T) {
	svc := &mocks.MockService{
		CreateNoteFunc: func(req domain.CreateNoteRequest) (domain.CreateNoteResponse, error) {
			return domain.CreateNoteResponse{}, fmt.Errorf("%w: type is required", domain.ErrInvalidInput)
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.CreateNoteRequest{Type: ""})
	r := httptest.NewRequest(http.MethodPost, "/api/notes", bytes.NewReader(body))
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.createNote(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "type is required")
}

func TestCreateNote_ServiceError_Returns500(t *testing.T) {
	svc := &mocks.MockService{
		CreateNoteFunc: func(req domain.CreateNoteRequest) (domain.CreateNoteResponse, error) {
			return domain.CreateNoteResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.CreateNoteRequest{Type: domain.NoteTypeText})
	r := httptest.NewRequest(http.MethodPost, "/api/notes", bytes.NewReader(body))
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.createNote(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NotContains(t, rr.Body.String(), "db error")
}

// --- getNotes ---

func TestGetNotes_Success(t *testing.T) {
	svc := &mocks.MockService{
		GetNotesFunc: func(req domain.GetNotesRequest) (domain.GetNotesResponse, error) {
			return domain.GetNotesResponse{Notes: []domain.Note{{Title: "Note1"}}}, nil
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/api/notes", nil)
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.getNotes(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Note1")
}

func TestGetNotes_NoUser(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodGet, "/api/notes", nil)
	rr := httptest.NewRecorder()

	h.getNotes(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestGetNotes_ServiceError_Returns500(t *testing.T) {
	svc := &mocks.MockService{
		GetNotesFunc: func(req domain.GetNotesRequest) (domain.GetNotesResponse, error) {
			return domain.GetNotesResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/api/notes", nil)
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.getNotes(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NotContains(t, rr.Body.String(), "db error")
}

// --- updateNote ---

func TestUpdateNote_Success(t *testing.T) {
	svc := &mocks.MockService{
		UpdateNoteFunc: func(req domain.UpdateNoteRequest) (domain.UpdateNoteResponse, error) {
			return domain.UpdateNoteResponse{ID: req.ID}, nil
		},
	}
	h := testHandler(svc)
	title := "Updated"
	body, _ := json.Marshal(domain.UpdateNoteRequest{Title: &title})
	r := httptest.NewRequest(http.MethodPatch, "/api/notes/5", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.updateNote(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"id":5`)
}

func TestUpdateNote_NoUser(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPatch, "/api/notes/5", nil)
	rr := httptest.NewRecorder()

	h.updateNote(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestUpdateNote_InvalidID(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPatch, "/api/notes/abc", nil)
	r = withUserAndChi(r, 1, "id", "abc")
	rr := httptest.NewRecorder()

	h.updateNote(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateNote_BadBody(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPatch, "/api/notes/5", bytes.NewReader([]byte("bad")))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.updateNote(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateNote_ValidationError_Returns400(t *testing.T) {
	svc := &mocks.MockService{
		UpdateNoteFunc: func(req domain.UpdateNoteRequest) (domain.UpdateNoteResponse, error) {
			return domain.UpdateNoteResponse{}, fmt.Errorf("%w: type must be one of text, checklist, bullet", domain.ErrInvalidInput)
		},
	}
	h := testHandler(svc)
	title := "x"
	body, _ := json.Marshal(domain.UpdateNoteRequest{Title: &title})
	r := httptest.NewRequest(http.MethodPatch, "/api/notes/5", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.updateNote(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "type must be one of")
}

func TestUpdateNote_NotFound(t *testing.T) {
	svc := &mocks.MockService{
		UpdateNoteFunc: func(req domain.UpdateNoteRequest) (domain.UpdateNoteResponse, error) {
			return domain.UpdateNoteResponse{}, gorm.ErrRecordNotFound
		},
	}
	h := testHandler(svc)
	title := "x"
	body, _ := json.Marshal(domain.UpdateNoteRequest{Title: &title})
	r := httptest.NewRequest(http.MethodPatch, "/api/notes/999", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "999")
	rr := httptest.NewRecorder()

	h.updateNote(rr, r)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestUpdateNote_ServiceError_Returns500(t *testing.T) {
	svc := &mocks.MockService{
		UpdateNoteFunc: func(req domain.UpdateNoteRequest) (domain.UpdateNoteResponse, error) {
			return domain.UpdateNoteResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	title := "x"
	body, _ := json.Marshal(domain.UpdateNoteRequest{Title: &title})
	r := httptest.NewRequest(http.MethodPatch, "/api/notes/5", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.updateNote(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NotContains(t, rr.Body.String(), "db error")
}

// --- deleteNote ---

func TestDeleteNote_Success(t *testing.T) {
	svc := &mocks.MockService{
		DeleteNoteFunc: func(req domain.DeleteNoteRequest) (domain.DeleteNoteResponse, error) {
			return domain.DeleteNoteResponse{ID: req.ID}, nil
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodDelete, "/api/notes/7", nil)
	r = withUserAndChi(r, 1, "id", "7")
	rr := httptest.NewRecorder()

	h.deleteNote(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"id":7`)
}

func TestDeleteNote_NoUser(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodDelete, "/api/notes/7", nil)
	rr := httptest.NewRecorder()

	h.deleteNote(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestDeleteNote_InvalidID(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodDelete, "/api/notes/abc", nil)
	r = withUserAndChi(r, 1, "id", "abc")
	rr := httptest.NewRecorder()

	h.deleteNote(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeleteNote_NotFound(t *testing.T) {
	svc := &mocks.MockService{
		DeleteNoteFunc: func(req domain.DeleteNoteRequest) (domain.DeleteNoteResponse, error) {
			return domain.DeleteNoteResponse{}, gorm.ErrRecordNotFound
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodDelete, "/api/notes/99", nil)
	r = withUserAndChi(r, 1, "id", "99")
	rr := httptest.NewRecorder()

	h.deleteNote(rr, r)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDeleteNote_ServiceError_Returns500(t *testing.T) {
	svc := &mocks.MockService{
		DeleteNoteFunc: func(req domain.DeleteNoteRequest) (domain.DeleteNoteResponse, error) {
			return domain.DeleteNoteResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodDelete, "/api/notes/5", nil)
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.deleteNote(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NotContains(t, rr.Body.String(), "db error")
}

// --- createNoteItem ---

func TestCreateNoteItem_Success(t *testing.T) {
	svc := &mocks.MockService{
		CreateNoteItemFunc: func(req domain.CreateNoteItemRequest) (domain.CreateNoteItemResponse, error) {
			return domain.CreateNoteItemResponse{ID: 10, NoteID: req.NoteID, Content: req.Content}, nil
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.CreateNoteItemRequest{Content: "Buy milk", Position: 0})
	r := httptest.NewRequest(http.MethodPost, "/api/notes/3/items", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "3")
	rr := httptest.NewRecorder()

	h.createNoteItem(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"id":10`)
}

func TestCreateNoteItem_NoUser(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPost, "/api/notes/3/items", nil)
	rr := httptest.NewRecorder()

	h.createNoteItem(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestCreateNoteItem_InvalidNoteID(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPost, "/api/notes/abc/items", nil)
	r = withUserAndChi(r, 1, "id", "abc")
	rr := httptest.NewRecorder()

	h.createNoteItem(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateNoteItem_BadBody(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPost, "/api/notes/3/items", bytes.NewReader([]byte("bad")))
	r = withUserAndChi(r, 1, "id", "3")
	rr := httptest.NewRecorder()

	h.createNoteItem(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateNoteItem_ValidationError_Returns400(t *testing.T) {
	svc := &mocks.MockService{
		CreateNoteItemFunc: func(req domain.CreateNoteItemRequest) (domain.CreateNoteItemResponse, error) {
			return domain.CreateNoteItemResponse{}, fmt.Errorf("%w: content is required", domain.ErrInvalidInput)
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.CreateNoteItemRequest{Content: ""})
	r := httptest.NewRequest(http.MethodPost, "/api/notes/3/items", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "3")
	rr := httptest.NewRecorder()

	h.createNoteItem(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "content is required")
}

func TestCreateNoteItem_NoteNotFound_Returns404(t *testing.T) {
	svc := &mocks.MockService{
		CreateNoteItemFunc: func(req domain.CreateNoteItemRequest) (domain.CreateNoteItemResponse, error) {
			return domain.CreateNoteItemResponse{}, gorm.ErrRecordNotFound
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.CreateNoteItemRequest{Content: "Buy milk"})
	r := httptest.NewRequest(http.MethodPost, "/api/notes/999/items", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "999")
	rr := httptest.NewRecorder()

	h.createNoteItem(rr, r)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestCreateNoteItem_ServiceError_Returns500(t *testing.T) {
	svc := &mocks.MockService{
		CreateNoteItemFunc: func(req domain.CreateNoteItemRequest) (domain.CreateNoteItemResponse, error) {
			return domain.CreateNoteItemResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.CreateNoteItemRequest{Content: "Buy milk"})
	r := httptest.NewRequest(http.MethodPost, "/api/notes/3/items", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "3")
	rr := httptest.NewRecorder()

	h.createNoteItem(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NotContains(t, rr.Body.String(), "db error")
}

// --- updateNoteItem ---

func TestUpdateNoteItem_Success(t *testing.T) {
	svc := &mocks.MockService{
		UpdateNoteItemFunc: func(req domain.UpdateNoteItemRequest) (domain.UpdateNoteItemResponse, error) {
			return domain.UpdateNoteItemResponse{ID: req.ID}, nil
		},
	}
	h := testHandler(svc)
	content := "Updated"
	body, _ := json.Marshal(domain.UpdateNoteItemRequest{Content: &content})
	r := httptest.NewRequest(http.MethodPatch, "/api/notes/3/items/10", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "3")
	r = withChi(r, "itemId", "10")
	rr := httptest.NewRecorder()

	h.updateNoteItem(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"id":10`)
}

func TestUpdateNoteItem_NoUser(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPatch, "/api/notes/3/items/10", nil)
	rr := httptest.NewRecorder()

	h.updateNoteItem(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestUpdateNoteItem_InvalidNoteID(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPatch, "/api/notes/abc/items/10", nil)
	r = withUserAndChi(r, 1, "id", "abc")
	r = withChi(r, "itemId", "10")
	rr := httptest.NewRecorder()

	h.updateNoteItem(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateNoteItem_InvalidItemID(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPatch, "/api/notes/3/items/abc", nil)
	r = withUserAndChi(r, 1, "id", "3")
	r = withChi(r, "itemId", "abc")
	rr := httptest.NewRecorder()

	h.updateNoteItem(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateNoteItem_NotFound(t *testing.T) {
	svc := &mocks.MockService{
		UpdateNoteItemFunc: func(req domain.UpdateNoteItemRequest) (domain.UpdateNoteItemResponse, error) {
			return domain.UpdateNoteItemResponse{}, gorm.ErrRecordNotFound
		},
	}
	h := testHandler(svc)
	content := "x"
	body, _ := json.Marshal(domain.UpdateNoteItemRequest{Content: &content})
	r := httptest.NewRequest(http.MethodPatch, "/api/notes/3/items/99", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "3")
	r = withChi(r, "itemId", "99")
	rr := httptest.NewRecorder()

	h.updateNoteItem(rr, r)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestUpdateNoteItem_ServiceError_Returns500(t *testing.T) {
	svc := &mocks.MockService{
		UpdateNoteItemFunc: func(req domain.UpdateNoteItemRequest) (domain.UpdateNoteItemResponse, error) {
			return domain.UpdateNoteItemResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	content := "x"
	body, _ := json.Marshal(domain.UpdateNoteItemRequest{Content: &content})
	r := httptest.NewRequest(http.MethodPatch, "/api/notes/3/items/10", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "3")
	r = withChi(r, "itemId", "10")
	rr := httptest.NewRecorder()

	h.updateNoteItem(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NotContains(t, rr.Body.String(), "db error")
}

// --- deleteNoteItem ---

func TestDeleteNoteItem_Success(t *testing.T) {
	svc := &mocks.MockService{
		DeleteNoteItemFunc: func(req domain.DeleteNoteItemRequest) (domain.DeleteNoteItemResponse, error) {
			return domain.DeleteNoteItemResponse{ID: req.ID}, nil
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodDelete, "/api/notes/3/items/10", nil)
	r = withUserAndChi(r, 1, "id", "3")
	r = withChi(r, "itemId", "10")
	rr := httptest.NewRecorder()

	h.deleteNoteItem(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"id":10`)
}

func TestDeleteNoteItem_NoUser(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodDelete, "/api/notes/3/items/10", nil)
	rr := httptest.NewRecorder()

	h.deleteNoteItem(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestDeleteNoteItem_NotFound(t *testing.T) {
	svc := &mocks.MockService{
		DeleteNoteItemFunc: func(req domain.DeleteNoteItemRequest) (domain.DeleteNoteItemResponse, error) {
			return domain.DeleteNoteItemResponse{}, gorm.ErrRecordNotFound
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodDelete, "/api/notes/3/items/99", nil)
	r = withUserAndChi(r, 1, "id", "3")
	r = withChi(r, "itemId", "99")
	rr := httptest.NewRecorder()

	h.deleteNoteItem(rr, r)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDeleteNoteItem_ServiceError_Returns500(t *testing.T) {
	svc := &mocks.MockService{
		DeleteNoteItemFunc: func(req domain.DeleteNoteItemRequest) (domain.DeleteNoteItemResponse, error) {
			return domain.DeleteNoteItemResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodDelete, "/api/notes/3/items/10", nil)
	r = withUserAndChi(r, 1, "id", "3")
	r = withChi(r, "itemId", "10")
	rr := httptest.NewRecorder()

	h.deleteNoteItem(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NotContains(t, rr.Body.String(), "db error")
}

// --- reorderNoteItems ---

func TestReorderNoteItems_Success(t *testing.T) {
	svc := &mocks.MockService{
		ReorderNoteItemsFunc: func(req domain.ReorderNoteItemsRequest) (domain.ReorderNoteItemsResponse, error) {
			return domain.ReorderNoteItemsResponse{NoteID: req.NoteID}, nil
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.ReorderNoteItemsRequest{ItemIDs: []uint{3, 1, 2}})
	r := httptest.NewRequest(http.MethodPut, "/api/notes/5/items/reorder", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.reorderNoteItems(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"noteId":5`)
}

func TestReorderNoteItems_NoUser(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPut, "/api/notes/5/items/reorder", nil)
	rr := httptest.NewRecorder()

	h.reorderNoteItems(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestReorderNoteItems_InvalidNoteID(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPut, "/api/notes/abc/items/reorder", nil)
	r = withUserAndChi(r, 1, "id", "abc")
	rr := httptest.NewRecorder()

	h.reorderNoteItems(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestReorderNoteItems_ValidationError_Returns400(t *testing.T) {
	svc := &mocks.MockService{
		ReorderNoteItemsFunc: func(req domain.ReorderNoteItemsRequest) (domain.ReorderNoteItemsResponse, error) {
			return domain.ReorderNoteItemsResponse{}, fmt.Errorf("%w: some item IDs do not belong to this note", domain.ErrInvalidInput)
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.ReorderNoteItemsRequest{ItemIDs: []uint{999}})
	r := httptest.NewRequest(http.MethodPut, "/api/notes/5/items/reorder", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.reorderNoteItems(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "do not belong")
}

func TestReorderNoteItems_NotFound(t *testing.T) {
	svc := &mocks.MockService{
		ReorderNoteItemsFunc: func(req domain.ReorderNoteItemsRequest) (domain.ReorderNoteItemsResponse, error) {
			return domain.ReorderNoteItemsResponse{}, gorm.ErrRecordNotFound
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.ReorderNoteItemsRequest{ItemIDs: []uint{1}})
	r := httptest.NewRequest(http.MethodPut, "/api/notes/999/items/reorder", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "999")
	rr := httptest.NewRecorder()

	h.reorderNoteItems(rr, r)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestReorderNoteItems_ServiceError_Returns500(t *testing.T) {
	svc := &mocks.MockService{
		ReorderNoteItemsFunc: func(req domain.ReorderNoteItemsRequest) (domain.ReorderNoteItemsResponse, error) {
			return domain.ReorderNoteItemsResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.ReorderNoteItemsRequest{ItemIDs: []uint{1}})
	r := httptest.NewRequest(http.MethodPut, "/api/notes/5/items/reorder", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.reorderNoteItems(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NotContains(t, rr.Body.String(), "db error")
}
