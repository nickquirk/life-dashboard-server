package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func (h *Handler) getProjectGroups(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	projectID, ok := h.parseIDParam(w, r, "id", "project", userID)
	if !ok {
		return
	}

	resp, err := h.Service.GetGroups(domain.GetGroupsRequest{UserID: userID, ProjectID: projectID})
	if err != nil {
		h.respondWithServiceError(w, err, "Project not found", "Failed to fetch groups", "userID", userID, "projectID", projectID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "projectID", projectID)
}

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	actionID, ok := h.parseIDParam(w, r, "id", "action", userID)
	if !ok {
		return
	}

	var req domain.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.ID = actionID
	req.UserID = userID

	resp, err := h.Service.CreateGroup(req)
	if err != nil {
		h.respondWithServiceError(w, err, "Action not found", "Failed to create group", "userID", userID, "actionID", actionID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "actionID", actionID)
}

func (h *Handler) setActionGroup(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	actionID, ok := h.parseIDParam(w, r, "id", "action", userID)
	if !ok {
		return
	}

	var req domain.SetActionGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.ID = actionID
	req.UserID = userID

	resp, err := h.Service.SetActionGroup(req)
	if err != nil {
		h.respondWithServiceError(w, err, "Action or group not found", "Failed to set action group", "userID", userID, "actionID", actionID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "actionID", actionID)
}

func (h *Handler) updateGroup(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	groupID, ok := h.parseIDParam(w, r, "id", "group", userID)
	if !ok {
		return
	}

	var req domain.UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.ID = groupID
	req.UserID = userID

	resp, err := h.Service.UpdateGroup(req)
	if err != nil {
		h.respondWithServiceError(w, err, "Group not found", "Failed to update group", "userID", userID, "groupID", groupID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "groupID", groupID)
}

func (h *Handler) ungroup(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}
	groupID, ok := h.parseIDParam(w, r, "id", "group", userID)
	if !ok {
		return
	}

	resp, err := h.Service.Ungroup(domain.UngroupRequest{ID: groupID, UserID: userID})
	if err != nil {
		h.respondWithServiceError(w, err, "Group not found", "Failed to ungroup", "userID", userID, "groupID", groupID)
		return
	}
	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "groupID", groupID)
}
