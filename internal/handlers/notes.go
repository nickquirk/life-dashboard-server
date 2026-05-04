package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

// --- Notes ---

func (h *Handler) createNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	var req domain.CreateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}

	req.UserID = userID

	resp, err := h.Service.CreateNote(req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			h.respondWithError(w, err.Error(), err, http.StatusBadRequest, "userID", userID)
			return
		}
		h.respondWithError(w, "Failed to create note", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}

func (h *Handler) getNotes(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	resp, err := h.Service.GetNotes(domain.GetNotesRequest{UserID: userID})
	if err != nil {
		h.respondWithError(w, "Failed to fetch notes", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}

func (h *Handler) updateNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	parsedID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.respondWithError(w, "Invalid note ID format", err, http.StatusBadRequest, "userID", userID)
		return
	}
	noteID := uint(parsedID)

	var req domain.UpdateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}

	req.ID = noteID
	req.UserID = userID

	resp, err := h.Service.UpdateNote(req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			h.respondWithError(w, err.Error(), err, http.StatusBadRequest, "userID", userID, "noteID", noteID)
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondWithError(w, "Note not found", err, http.StatusNotFound, "userID", userID, "noteID", noteID)
			return
		}
		h.respondWithError(w, "Failed to update note", err, http.StatusInternalServerError, "userID", userID, "noteID", noteID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "noteID", noteID)
}

func (h *Handler) deleteNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "Unauthorized", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	noteID, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondWithError(w, "Invalid note ID", err, http.StatusBadRequest, "userID", userID)
		return
	}

	resp, err := h.Service.DeleteNote(domain.DeleteNoteRequest{ID: uint(noteID), UserID: userID})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondWithError(w, "Note not found", err, http.StatusNotFound, "userID", userID, "noteID", noteID)
			return
		}
		h.respondWithError(w, "Failed to delete note", err, http.StatusInternalServerError, "userID", userID, "noteID", noteID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "noteID", noteID)
}

// --- Note Items ---

func (h *Handler) createNoteItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	parsedID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.respondWithError(w, "Invalid note ID format", err, http.StatusBadRequest, "userID", userID)
		return
	}
	noteID := uint(parsedID)

	var req domain.CreateNoteItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}

	req.UserID = userID
	req.NoteID = noteID

	resp, err := h.Service.CreateNoteItem(req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			h.respondWithError(w, err.Error(), err, http.StatusBadRequest, "userID", userID, "noteID", noteID)
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondWithError(w, "Note not found", err, http.StatusNotFound, "userID", userID, "noteID", noteID)
			return
		}
		h.respondWithError(w, "Failed to create note item", err, http.StatusInternalServerError, "userID", userID, "noteID", noteID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "noteID", noteID)
}

func (h *Handler) updateNoteItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	parsedNoteID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.respondWithError(w, "Invalid note ID format", err, http.StatusBadRequest, "userID", userID)
		return
	}
	noteID := uint(parsedNoteID)

	itemIDStr := chi.URLParam(r, "itemId")
	parsedItemID, err := strconv.ParseUint(itemIDStr, 10, 32)
	if err != nil {
		h.respondWithError(w, "Invalid item ID format", err, http.StatusBadRequest, "userID", userID)
		return
	}
	itemID := uint(parsedItemID)

	var req domain.UpdateNoteItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}

	req.UserID = userID
	req.NoteID = noteID
	req.ID = itemID

	resp, err := h.Service.UpdateNoteItem(req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			h.respondWithError(w, err.Error(), err, http.StatusBadRequest, "userID", userID, "noteID", noteID, "itemID", itemID)
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondWithError(w, "Note item not found", err, http.StatusNotFound, "userID", userID, "noteID", noteID, "itemID", itemID)
			return
		}
		h.respondWithError(w, "Failed to update note item", err, http.StatusInternalServerError, "userID", userID, "noteID", noteID, "itemID", itemID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "noteID", noteID, "itemID", itemID)
}

func (h *Handler) deleteNoteItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "Unauthorized", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	noteID, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondWithError(w, "Invalid note ID", err, http.StatusBadRequest, "userID", userID)
		return
	}

	itemIDStr := chi.URLParam(r, "itemId")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		h.respondWithError(w, "Invalid item ID", err, http.StatusBadRequest, "userID", userID)
		return
	}

	resp, err := h.Service.DeleteNoteItem(domain.DeleteNoteItemRequest{
		ID:     uint(itemID),
		UserID: userID,
		NoteID: uint(noteID),
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondWithError(w, "Note item not found", err, http.StatusNotFound, "userID", userID, "noteID", noteID, "itemID", itemID)
			return
		}
		h.respondWithError(w, "Failed to delete note item", err, http.StatusInternalServerError, "userID", userID, "noteID", noteID, "itemID", itemID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "noteID", noteID, "itemID", itemID)
}

func (h *Handler) reorderNoteItems(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	parsedID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.respondWithError(w, "Invalid note ID format", err, http.StatusBadRequest, "userID", userID)
		return
	}
	noteID := uint(parsedID)

	var req domain.ReorderNoteItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}

	req.UserID = userID
	req.NoteID = noteID

	resp, err := h.Service.ReorderNoteItems(req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			h.respondWithError(w, err.Error(), err, http.StatusBadRequest, "userID", userID, "noteID", noteID)
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondWithError(w, "Note not found", err, http.StatusNotFound, "userID", userID, "noteID", noteID)
			return
		}
		h.respondWithError(w, "Failed to reorder note items", err, http.StatusInternalServerError, "userID", userID, "noteID", noteID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "noteID", noteID)
}
