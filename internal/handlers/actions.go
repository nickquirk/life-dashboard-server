package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

// getActions serves ?stage=ready, the Plan sidebar's ready actions across
// active projects. stage=ready is required.
func (h *Handler) getActions(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	if stage := r.URL.Query().Get("stage"); domain.Stage(stage) != domain.StageReady {
		h.respondWithError(w, "stage=ready is required", fmt.Errorf("unsupported stage %q", stage), http.StatusBadRequest, "userID", userID)
		return
	}
	resp, err := h.Service.GetReadyActions(domain.GetReadyActionsRequest{UserID: userID})
	if err != nil {
		h.respondWithError(w, "Failed to fetch ready actions", err, http.StatusInternalServerError, "userID", userID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}

// parseRange reads the calendar range ?start=…&end=… (RFC3339). On failure it
// has already responded 400.
func (h *Handler) parseRange(w http.ResponseWriter, r *http.Request, userID uint) (time.Time, time.Time, bool) {
	q := r.URL.Query()
	startStr, endStr := q.Get("start"), q.Get("end")
	if startStr == "" || endStr == "" {
		h.respondWithError(w, "Missing start or end query parameters", fmt.Errorf("missing start or end query parameter"), http.StatusBadRequest, "userID", userID)
		return time.Time{}, time.Time{}, false
	}
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		h.respondWithError(w, "Invalid start format, expected RFC3339", err, http.StatusBadRequest, "userID", userID)
		return time.Time{}, time.Time{}, false
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		h.respondWithError(w, "Invalid end format, expected RFC3339", err, http.StatusBadRequest, "userID", userID)
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

func (h *Handler) updateAction(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	actionID, ok := h.parseIDParam(w, r, "id", "action", userID)
	if !ok {
		return
	}

	var req domain.UpdateActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.ID = actionID
	req.UserID = userID

	resp, err := h.Service.UpdateAction(req)
	if err != nil {
		h.respondWithServiceError(w, err, "Action not found", "Failed to update action", "userID", userID, "actionID", actionID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "actionID", actionID)
}

func (h *Handler) changeActionStage(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	actionID, ok := h.parseIDParam(w, r, "id", "action", userID)
	if !ok {
		return
	}

	var req domain.ChangeStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.ID = actionID
	req.UserID = userID

	resp, err := h.Service.ChangeActionStage(req)
	if err != nil {
		h.respondWithServiceError(w, err, "Action not found", "Failed to change action stage", "userID", userID, "actionID", actionID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "actionID", actionID)
}

func (h *Handler) completeAction(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	actionID, ok := h.parseIDParam(w, r, "id", "action", userID)
	if !ok {
		return
	}

	// The body is optional: {"workSessionId": n} ticks it off inside that Work
	// session. No body is a plain complete
	req := domain.CompleteActionRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.ID, req.UserID = actionID, userID

	resp, err := h.Service.CompleteAction(req)
	if err != nil {
		h.respondWithServiceError(w, err, "Action or Work session not found", "Failed to complete action", "userID", userID, "actionID", actionID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "actionID", actionID)
}

func (h *Handler) reopenAction(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	actionID, ok := h.parseIDParam(w, r, "id", "action", userID)
	if !ok {
		return
	}

	resp, err := h.Service.ReopenAction(domain.ReopenActionRequest{ID: actionID, UserID: userID})
	if err != nil {
		h.respondWithServiceError(w, err, "Action not found", "Failed to reopen action", "userID", userID, "actionID", actionID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "actionID", actionID)
}

func (h *Handler) breakDownAction(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	actionID, ok := h.parseIDParam(w, r, "id", "action", userID)
	if !ok {
		return
	}

	var req domain.BreakDownActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.ID = actionID
	req.UserID = userID

	resp, err := h.Service.BreakDownAction(req)
	if err != nil {
		h.respondWithServiceError(w, err, "Action not found", "Failed to break down action", "userID", userID, "actionID", actionID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "actionID", actionID)
}

func (h *Handler) deleteAction(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	actionID, ok := h.parseIDParam(w, r, "id", "action", userID)
	if !ok {
		return
	}

	resp, err := h.Service.DeleteAction(domain.DeleteActionRequest{ID: actionID, UserID: userID})
	if err != nil {
		h.respondWithServiceError(w, err, "Action not found", "Failed to delete action", "userID", userID, "actionID", actionID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "actionID", actionID)
}
