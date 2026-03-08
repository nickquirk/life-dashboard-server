package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/gorm"
)

func (h *Handler) getScratchpad(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	date := r.URL.Query().Get("date") // Expects ?date=YYYY-MM-DD
	if date == "" {
		http.Error(w, "Missing date parameter", http.StatusBadRequest)
		return
	}

	// Validate date format exactly matches YYYY-MM-DD
	if _, err := time.Parse(time.DateOnly, date); err != nil {
		http.Error(w, "Invalid date format, expected YYYY-MM-DD", http.StatusBadRequest)
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
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(domain.GetScratchpadResponse{Content: ""})
			return
		}
		http.Error(w, "Failed to fetch scratchpad", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) upsertScratchpad(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req domain.UpsertScratchpadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate date format exactly matches YYYY-MM-DD
	if _, err := time.Parse(time.DateOnly, req.Date); err != nil {
		http.Error(w, "Invalid date format, expected YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	// Attach the secured UserID from the JWT context
	req.UserID = userID

	resp, err := h.Service.UpsertScratchpad(req)
	if err != nil {
		http.Error(w, "Failed to save scratchpad", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
