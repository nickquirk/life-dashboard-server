package handlers

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"os"
)

func (h *Handler) handleGlobalSync(w http.ResponseWriter, r *http.Request) {
	// Secondary check — shared secret (primary authn is OIDC middleware)
	secret := os.Getenv("SCHEDULER_SECRET")
	header := r.Header.Get("X-Scheduler-Secret")
	if subtle.ConstantTimeCompare([]byte(header), []byte(secret)) != 1 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Execute the Job
	// Cloud Run can kill requests after 60 mins, but usually keep this shorter
	err := h.Service.SyncAllUsers(r.Context())
	if err != nil {
		slog.Error("global sync failed", "error", err)
		http.Error(w, "Sync failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Sync complete"))
}
