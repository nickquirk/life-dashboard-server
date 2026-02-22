package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

// GET /api/events
func (h *Handler) getCalendarEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// refactor into helper getUserIDFromConfig
	userID, ok := ctx.Value(UserIDKey).(uint)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Parse Query Params (start/end)
	// Example: ?start=2024-01-01T00:00:00Z&end=2024-01-07T00:00:00Z
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		http.Error(w, "Missing required query params: start and end", http.StatusBadRequest)
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		http.Error(w, "Invalid start param, expected RFC3339 format", http.StatusBadRequest)
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		http.Error(w, "Invalid end param, expected RFC3339 format", http.StatusBadRequest)
		return
	}

	dto := domain.GetCalendarEventsRequest{
		UserID: userID,
		Start:  start,
		End:    end,
	}

	resp, err := h.Service.GetCalendarEvents(ctx, dto)
	if err != nil {
		http.Error(w, "Failed to fetch calendar events", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
