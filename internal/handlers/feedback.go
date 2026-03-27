package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func (h *Handler) createFeedback(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	var req domain.CreateFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.UserID = userID

	resp, err := h.Service.CreateFeedback(req)
	if err != nil {
		h.respondWithError(w, "Failed to create feedback", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
