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

// parseIDParam reads a uint URL param, responding 400 on failure.
func (h *Handler) parseIDParam(w http.ResponseWriter, r *http.Request, name, label string, userID uint) (uint, bool) {
	parsed, err := strconv.ParseUint(chi.URLParam(r, name), 10, 32)
	if err != nil {
		h.respondWithError(w, "Invalid "+label+" ID format", err, http.StatusBadRequest, "userID", userID)
		return 0, false
	}
	return uint(parsed), true
}

// respondWithServiceError maps the service's sentinel errors to HTTP:
// ErrInvalidInput → 400 with its message, ErrRecordNotFound → 404, else 500.
func (h *Handler) respondWithServiceError(w http.ResponseWriter, err error, notFoundMsg, failMsg string, args ...any) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		h.respondWithError(w, err.Error(), err, http.StatusBadRequest, args...)
	case errors.Is(err, gorm.ErrRecordNotFound):
		h.respondWithError(w, notFoundMsg, err, http.StatusNotFound, args...)
	default:
		h.respondWithError(w, failMsg, err, http.StatusInternalServerError, args...)
	}
}

// --- Projects ---

func (h *Handler) getProjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	archived := r.URL.Query().Get("archived") == "true"

	resp, err := h.Service.GetProjects(domain.GetProjectsRequest{UserID: userID, Archived: archived})
	if err != nil {
		h.respondWithError(w, "Failed to fetch projects", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	var req domain.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.UserID = userID

	resp, err := h.Service.CreateProject(req)
	if err != nil {
		h.respondWithServiceError(w, err, "Project not found", "Failed to create project", "userID", userID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}

func (h *Handler) updateProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	projectID, ok := h.parseIDParam(w, r, "id", "project", userID)
	if !ok {
		return
	}

	var req domain.UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.ID = projectID
	req.UserID = userID

	resp, err := h.Service.UpdateProject(req)
	if err != nil {
		h.respondWithServiceError(w, err, "Project not found", "Failed to update project", "userID", userID, "projectID", projectID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "projectID", projectID)
}

func (h *Handler) deleteProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	projectID, ok := h.parseIDParam(w, r, "id", "project", userID)
	if !ok {
		return
	}

	resp, err := h.Service.DeleteProject(domain.DeleteProjectRequest{ID: projectID, UserID: userID})
	if err != nil {
		h.respondWithServiceError(w, err, "Project not found", "Failed to delete project", "userID", userID, "projectID", projectID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "projectID", projectID)
}

// --- Project board ---

func (h *Handler) getProjectActions(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	projectID, ok := h.parseIDParam(w, r, "id", "project", userID)
	if !ok {
		return
	}

	resp, err := h.Service.GetActions(domain.GetActionsRequest{UserID: userID, ProjectID: projectID})
	if err != nil {
		h.respondWithServiceError(w, err, "Project not found", "Failed to fetch actions", "userID", userID, "projectID", projectID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "projectID", projectID)
}

func (h *Handler) createAction(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	projectID, ok := h.parseIDParam(w, r, "id", "project", userID)
	if !ok {
		return
	}

	var req domain.CreateActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.UserID = userID
	req.ProjectID = projectID

	resp, err := h.Service.CreateAction(req)
	if err != nil {
		h.respondWithServiceError(w, err, "Project not found", "Failed to create action", "userID", userID, "projectID", projectID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "projectID", projectID)
}

func (h *Handler) reorderActions(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	projectID, ok := h.parseIDParam(w, r, "id", "project", userID)
	if !ok {
		return
	}

	var req domain.ReorderActionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.UserID = userID
	req.ProjectID = projectID

	resp, err := h.Service.ReorderActions(req)
	if err != nil {
		h.respondWithServiceError(w, err, "Project not found", "Failed to reorder actions", "userID", userID, "projectID", projectID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "projectID", projectID)
}
