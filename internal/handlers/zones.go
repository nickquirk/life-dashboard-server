package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

func (h *Handler) createZone(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var req domain.CreateZoneRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.UserID = userID

	resp, err := h.Service.CreateZone(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) getZones(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var req = domain.GetZonesRequest{
		UserID: userID,
	}

	resp, err := h.Service.GetZones(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) deleteZone(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract zone ID from URL /api/zones/{id}
	idStr := chi.URLParam(r, "id")
	zoneID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid zone ID", http.StatusBadRequest)
		return
	}

	req := domain.DeleteZonesRequest{
		ID:     uint(zoneID),
		UserID: userID,
	}

	resp, err := h.Service.DeleteZone(req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "Zone not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

}
