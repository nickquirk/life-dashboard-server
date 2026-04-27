package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

// --- Routines ---

func (h *Handler) createRoutine(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	var req domain.CreateRoutineRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}

	req.UserID = userID

	resp, err := h.Service.CreateRoutine(req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			h.respondWithError(w, err.Error(), err, http.StatusBadRequest, "userID", userID)
			return
		}
		// POss change to opaque error
		h.respondWithError(w, "Failed to create routine", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}

func (h *Handler) getRoutines(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	var req = domain.GetRoutineRequest{
		UserID: userID,
	}

	resp, err := h.Service.GetRoutines(req)
	if err != nil {
		h.respondWithError(w, "Failed to fetch routines", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}

func (h *Handler) updateRoutine(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	parsedID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.respondWithError(w, "Invalid routine ID format", err, http.StatusBadRequest, "userID", userID)
		return
	}
	routineID := uint(parsedID)

	var req domain.UpdateRoutineRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}

	req.ID = routineID
	req.UserID = userID

	resp, err := h.Service.UpdateRoutine(req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			h.respondWithError(w, err.Error(), err, http.StatusBadRequest, "userID", userID, "routineID", routineID)
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondWithError(w, "Routine not found", err, http.StatusNotFound, "userID", userID, "routineID", routineID)
			return
		}
		h.respondWithError(w, "Failed to update routine", err, http.StatusInternalServerError, "userID", userID, "routineID", routineID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "routineID", routineID)
}

func (h *Handler) deleteRoutine(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "Unauthorized", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	routineID, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondWithError(w, "Invalid routine ID", err, http.StatusBadRequest, "userID", userID)
		return
	}

	req := domain.DeleteRoutineRequest{
		ID:     uint(routineID),
		UserID: userID,
	}

	resp, err := h.Service.DeleteRoutine(req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondWithError(w, "Routine not found", err, http.StatusNotFound, "userID", userID, "routineID", routineID)
			return
		}
		h.respondWithError(w, "Failed to delete routine", err, http.StatusInternalServerError, "userID", userID, "routineID", routineID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "routineID", routineID)
}

// --- Routine Instances ---

func (h *Handler) createRoutineInstance(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	var req domain.CreateRoutineInstanceRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}

	req.UserID = userID

	resp, err := h.Service.CreateRoutineInstance(req)
	if err != nil {
		h.respondWithError(w, "Failed to create routine instance", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}

func (h *Handler) getRoutineInstances(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	// Read from URL parameters
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		h.respondWithError(w, "Missing start or end query parameters", fmt.Errorf("missing start or end query parameter"), http.StatusBadRequest, "userID", userID)
		return
	}

	// Parse into time.Time
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

	// Build request DTO
	req := domain.GetRoutineInstancesRequest{
		UserID: userID,
		Start:  start,
		End:    end,
	}

	// Fetch from service
	resp, err := h.Service.GetRoutineInstances(req)
	if err != nil {
		h.respondWithError(w, "Failed to fetch routine instances", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}

func (h *Handler) updateRoutineInstance(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	parsedID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.respondWithError(w, "Invalid instance ID format", err, http.StatusBadRequest, "userID", userID)
		return
	}
	instanceID := uint(parsedID)

	var req domain.UpdateRoutineInstanceRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}

	req.ID = instanceID
	req.UserID = userID

	resp, err := h.Service.UpdateRoutineInstance(req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondWithError(w, "Routine instance not found", err, http.StatusNotFound, "userID", userID, "instanceID", instanceID)
			return
		}
		h.respondWithError(w, "Failed to update routine instance", err, http.StatusInternalServerError, "userID", userID, "instanceID", instanceID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "instanceID", instanceID)
}

func (h *Handler) deleteRoutineInstance(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "Unauthorized", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	instanceID, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondWithError(w, "Invalid instance ID", err, http.StatusBadRequest, "userID", userID)
		return
	}

	req := domain.DeleteRoutineInstanceRequest{
		ID:     uint(instanceID),
		UserID: userID,
	}

	resp, err := h.Service.DeleteRoutineInstance(req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondWithError(w, "Routine instance not found", err, http.StatusNotFound, "userID", userID, "instanceID", instanceID)
			return
		}
		h.respondWithError(w, "Failed to delete routine instance", err, http.StatusInternalServerError, "userID", userID, "instanceID", instanceID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "instanceID", instanceID)
}
