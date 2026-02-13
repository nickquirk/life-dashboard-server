package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func (h *Handler) createZone(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// refactor into helper getUserIDFromConfig
	userID, ok := ctx.Value(UserIDKey).(uint)
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

	resp, err := h.Service.CreateZone(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
