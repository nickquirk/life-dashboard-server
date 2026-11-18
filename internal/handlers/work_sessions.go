package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

// getWorkSessions serves the calendar range ?start=…&end=… (RFC3339),
// including archived projects' Work sessions.
func (h *Handler) getWorkSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	start, end, ok := h.parseRange(w, r, userID)
	if !ok {
		return
	}

	resp, err := h.Service.GetWorkSessions(domain.GetWorkSessionsRequest{UserID: userID, Start: start, End: end})
	if err != nil {
		h.respondWithError(w, "Failed to fetch Work sessions", err, http.StatusInternalServerError, "userID", userID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}

func (h *Handler) createWorkSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	projectID, ok := h.parseIDParam(w, r, "id", "project", userID)
	if !ok {
		return
	}

	var req domain.CreateWorkSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.ProjectID = projectID
	req.UserID = userID

	resp, err := h.Service.CreateWorkSession(req)
	if err != nil {
		h.respondWithServiceError(w, err, "Project or group not found", "Failed to create Work session", "userID", userID, "projectID", projectID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "projectID", projectID)
}

func (h *Handler) updateWorkSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	id, ok := h.parseIDParam(w, r, "id", "work session", userID)
	if !ok {
		return
	}

	var req domain.UpdateWorkSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.ID = id
	req.UserID = userID

	resp, err := h.Service.UpdateWorkSession(req)
	if err != nil {
		h.respondWithServiceError(w, err, "Work session not found", "Failed to update Work session", "userID", userID, "workSessionID", id)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "workSessionID", id)
}

func (h *Handler) completeWorkSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	id, ok := h.parseIDParam(w, r, "id", "work session", userID)
	if !ok {
		return
	}

	resp, err := h.Service.CompleteWorkSession(domain.CompleteWorkSessionRequest{ID: id, UserID: userID})
	if err != nil {
		h.respondWithServiceError(w, err, "Work session not found", "Failed to complete Work session", "userID", userID, "workSessionID", id)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "workSessionID", id)
}

func (h *Handler) reopenWorkSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	id, ok := h.parseIDParam(w, r, "id", "work session", userID)
	if !ok {
		return
	}

	resp, err := h.Service.ReopenWorkSession(domain.ReopenWorkSessionRequest{ID: id, UserID: userID})
	if err != nil {
		h.respondWithServiceError(w, err, "Work session not found", "Failed to reopen Work session", "userID", userID, "workSessionID", id)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "workSessionID", id)
}

func (h *Handler) deleteWorkSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	id, ok := h.parseIDParam(w, r, "id", "work session", userID)
	if !ok {
		return
	}

	resp, err := h.Service.DeleteWorkSession(domain.DeleteWorkSessionRequest{ID: id, UserID: userID})
	if err != nil {
		h.respondWithServiceError(w, err, "Work session not found", "Failed to delete Work session", "userID", userID, "workSessionID", id)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "workSessionID", id)
}
