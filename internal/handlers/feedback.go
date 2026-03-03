package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func (h *Handler) createFeedback(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var req domain.CreateFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	req.UserID = userID

	resp, err := h.Service.CreateFeedback(req)
	if err != nil {
		http.Error(w, "Failed to create feedback", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
