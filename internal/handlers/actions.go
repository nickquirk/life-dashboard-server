package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

// getActions serves the two cross-project reads:
//
//	?stage=ready            Plan sidebar: ready actions across active projects
//	?start=…&end=… (RFC3339) calendar range, including archived projects
func (h *Handler) getActions(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	q := r.URL.Query()
	if stage := q.Get("stage"); stage != "" {
		if domain.Stage(stage) != domain.StageReady {
			h.respondWithError(w, "Only stage=ready is supported", fmt.Errorf("unsupported stage %q", stage), http.StatusBadRequest, "userID", userID)
			return
		}
		resp, err := h.Service.GetReadyActions(domain.GetReadyActionsRequest{UserID: userID})
		if err != nil {
			h.respondWithError(w, "Failed to fetch ready actions", err, http.StatusInternalServerError, "userID", userID)
			return
		}
		h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
		return
	}

	startStr, endStr := q.Get("start"), q.Get("end")
	if startStr == "" || endStr == "" {
		h.respondWithError(w, "Missing start or end query parameters", fmt.Errorf("missing start or end query parameter"), http.StatusBadRequest, "userID", userID)
		return
	}
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		h.respondWithError(w, "Invalid start format, expected RFC3339", err, http.StatusBadRequest, "userID", userID)
		return
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		h.respondWithError(w, "Invalid end format, expected RFC3339", err, http.StatusBadRequest, "userID", userID)
		return
	}

	resp, err := h.Service.GetScheduledActions(domain.GetScheduledActionsRequest{UserID: userID, Start: start, End: end})
	if err != nil {
		h.respondWithError(w, "Failed to fetch scheduled actions", err, http.StatusInternalServerError, "userID", userID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
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

func (h *Handler) scheduleAction(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	actionID, ok := h.parseIDParam(w, r, "id", "action", userID)
	if !ok {
		return
	}

	var req domain.ScheduleActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.ID = actionID
	req.UserID = userID

	resp, err := h.Service.ScheduleAction(req)
	if err != nil {
		h.respondWithServiceError(w, err, "Action not found", "Failed to schedule action", "userID", userID, "actionID", actionID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "actionID", actionID)
}

func (h *Handler) moveAction(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	actionID, ok := h.parseIDParam(w, r, "id", "action", userID)
	if !ok {
		return
	}

	var req domain.MoveActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.ID = actionID
	req.UserID = userID

	resp, err := h.Service.MoveAction(req)
	if err != nil {
		h.respondWithServiceError(w, err, "Action not found", "Failed to move action", "userID", userID, "actionID", actionID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "actionID", actionID)
}

func (h *Handler) unscheduleAction(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	actionID, ok := h.parseIDParam(w, r, "id", "action", userID)
	if !ok {
		return
	}

	resp, err := h.Service.UnscheduleAction(domain.UnscheduleActionRequest{ID: actionID, UserID: userID})
	if err != nil {
		h.respondWithServiceError(w, err, "Action not found", "Failed to unschedule action", "userID", userID, "actionID", actionID)
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

	resp, err := h.Service.CompleteAction(domain.CompleteActionRequest{ID: actionID, UserID: userID})
	if err != nil {
		h.respondWithServiceError(w, err, "Action not found", "Failed to complete action", "userID", userID, "actionID", actionID)
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
