package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

// GET /api/events
func (h *Handler) getCalendarEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	// Parse Query Params (start/end)
	// Example: ?start=2024-01-01T00:00:00Z&end=2024-01-07T00:00:00Z
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		h.respondWithError(w, "Missing required query params: start and end", fmt.Errorf("missing start or end query parameter"), http.StatusBadRequest, "userID", userID)
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		h.respondWithError(w, "Invalid start param, expected RFC3339 format", err, http.StatusBadRequest, "userID", userID)
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		h.respondWithError(w, "Invalid end param, expected RFC3339 format", err, http.StatusBadRequest, "userID", userID)
		return
	}

	dto := domain.GetCalendarEventsRequest{
		UserID: userID,
		Start:  start,
		End:    end,
	}

	resp, err := h.Service.GetCalendarEvents(ctx, dto)
	if err != nil {
		h.respondWithError(w, "Failed to fetch calendar events", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	h.respondWithJSON(w, http.StatusOK, resp, "userID", userID)
}
