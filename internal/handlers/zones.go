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

func (h *Handler) createZone(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	var req domain.CreateZoneRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}

	req.UserID = userID

	resp, err := h.Service.CreateZone(req)
	if err != nil {
		h.respondWithError(w, "Failed to create zone", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}

func (h *Handler) getZones(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	var req = domain.GetZonesRequest{
		UserID: userID,
	}

	resp, err := h.Service.GetZones(req)
	if err != nil {
		h.respondWithError(w, "Failed to fetch zones", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}

func (h *Handler) updateZone(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	zoneIDStr := chi.URLParam(r, "id")

	// Parse string to uint64, then cast to uint
	parsedID, err := strconv.ParseUint(zoneIDStr, 10, 32)
	if err != nil {
		h.respondWithError(w, "Invalid zone ID format", err, http.StatusBadRequest, "userID", userID)
		return
	}
	zoneID := uint(parsedID)

	var req domain.UpdateZoneRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}

	req.ID = zoneID
	req.UserID = userID

	resp, err := h.Service.UpdateZone(req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondWithError(w, "Zone not found", err, http.StatusNotFound, "userID", userID, "zoneID", zoneID)
			return
		}
		h.respondWithError(w, "Failed to update zone", err, http.StatusInternalServerError, "userID", userID, "zoneID", zoneID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "zoneID", zoneID)
}

func (h *Handler) deleteZone(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "Unauthorized", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	// Extract zone ID from URL /api/zones/{id}
	idStr := chi.URLParam(r, "id")
	zoneID, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondWithError(w, "Invalid zone ID", err, http.StatusBadRequest, "userID", userID)
		return
	}

	req := domain.DeleteZoneRequest{
		ID:     uint(zoneID),
		UserID: userID,
	}

	resp, err := h.Service.DeleteZone(req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.respondWithError(w, "Zone not found", err, http.StatusNotFound, "userID", userID, "zoneID", zoneID)
			return
		}
		h.respondWithError(w, "Failed to delete zone", err, http.StatusInternalServerError, "userID", userID, "zoneID", zoneID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID, "zoneID", zoneID)
}
