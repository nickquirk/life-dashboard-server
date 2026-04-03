package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// respondWithError logs the internal error to GCP and returns a safe HTTP response to the frontend.
func (h *Handler) respondWithError(w http.ResponseWriter, userMsg string, internalErr error, status int, args ...any) {
	// 1. Build the log attributes starting with the actual error
	logArgs := append([]any{"error", internalErr.Error()}, args...)

	// Log it so GCP captures it
	slog.Error(userMsg, logArgs...)

	// Return the sanitized message to the client
	http.Error(w, userMsg, status)
}

func (h *Handler) respondWithJSON(w http.ResponseWriter, status int, payload interface{}, args ...any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// Append any extra context (like userID) to the log arguments
		logArgs := append([]any{"error", err.Error()}, args...)
		slog.Error("failed to write JSON response", logArgs...)
	}
}
