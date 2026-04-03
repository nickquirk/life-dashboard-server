package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

func (h *Handler) getScratchpad(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "Unauthorized", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	date := r.URL.Query().Get("date") // Expects ?date=YYYY-MM-DD
	if date == "" {
		h.respondWithError(w, "Missing date parameter", fmt.Errorf("missing date query parameter"), http.StatusBadRequest, "userID", userID)
		return
	}

	// Validate date format exactly matches YYYY-MM-DD
	if _, err := time.Parse(time.DateOnly, date); err != nil {
		h.respondWithError(w, "Invalid date format, expected YYYY-MM-DD", err, http.StatusBadRequest, "userID", userID)
		return
	}

	req := domain.GetScratchpadRequest{
		UserID: userID,
		Date:   date,
	}

	resp, err := h.Service.GetScratchpad(req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return empty 200 instead of 404 for seamless frontend UX
			h.respondWithJSON(w, http.StatusOK, domain.GetScratchpadResponse{Content: ""}, "userID", userID)
			return
		}
		h.respondWithError(w, "Failed to fetch scratchpad", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}

func (h *Handler) upsertScratchpad(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "Unauthorized", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	var req domain.UpsertScratchpadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}

	// Validate date format exactly matches YYYY-MM-DD
	if _, err := time.Parse(time.DateOnly, req.Date); err != nil {
		h.respondWithError(w, "Invalid date format, expected YYYY-MM-DD", err, http.StatusBadRequest, "userID", userID)
		return
	}

	// Attach the secured UserID from the JWT context
	req.UserID = userID

	resp, err := h.Service.UpsertScratchpad(req)
	if err != nil {
		h.respondWithError(w, "Failed to save scratchpad", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}
