package handlers

import (
	"encoding/json"
	"errors"
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
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var req domain.CreateRoutineRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.UserID = userID

	resp, err := h.Service.CreateRoutine(req)
	if err != nil {
		http.Error(w, "Failed to create routine", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) getRoutines(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var req = domain.GetRoutineRequest{
		UserID: userID,
	}

	resp, err := h.Service.GetRoutines(req)
	if err != nil {
		http.Error(w, "Failed to fetch routines", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) updateRoutine(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	parsedID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid routine ID format", http.StatusBadRequest)
		return
	}
	routineID := uint(parsedID)

	var req domain.UpdateRoutineRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.ID = routineID
	req.UserID = userID

	resp, err := h.Service.UpdateRoutine(req)
	if err != nil {
		http.Error(w, "Failed to update routine", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) deleteRoutine(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	routineID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid routine ID", http.StatusBadRequest)
		return
	}

	req := domain.DeleteRoutineRequest{
		ID:     uint(routineID),
		UserID: userID,
	}

	resp, err := h.Service.DeleteRoutine(req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Routine not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete routine", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// --- Routine Instances ---

func (h *Handler) createRoutineInstance(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var req domain.CreateRoutineInstanceRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.UserID = userID

	resp, err := h.Service.CreateRoutineInstance(req)
	if err != nil {
		http.Error(w, "Failed to create routine instance", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) getRoutineInstances(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Read from URL parameters
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		http.Error(w, "Missing start or end query parameters", http.StatusBadRequest)
		return
	}

	// Parse into time.Time
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		http.Error(w, "Invalid start format, expected RFC3339", http.StatusBadRequest)
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		http.Error(w, "Invalid end format, expected RFC3339", http.StatusBadRequest)
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
		http.Error(w, "Failed to fetch routine instances", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) updateRoutineInstance(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	parsedID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid instance ID format", http.StatusBadRequest)
		return
	}
	instanceID := uint(parsedID)

	var req domain.UpdateRoutineInstanceRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.ID = instanceID
	req.UserID = userID

	resp, err := h.Service.UpdateRoutineInstance(req)
	if err != nil {
		http.Error(w, "Failed to update routine instance", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) deleteRoutineInstance(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	instanceID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid instance ID", http.StatusBadRequest)
		return
	}

	req := domain.DeleteRoutineInstanceRequest{
		ID:     uint(instanceID),
		UserID: userID,
	}

	resp, err := h.Service.DeleteRoutineInstance(req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Routine instance not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete routine instance", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
