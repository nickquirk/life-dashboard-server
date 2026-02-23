package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
)

// 1. The Trigger Handler (Called by Cloud Scheduler)
func (h *Handler) handleTriggerSync(w http.ResponseWriter, r *http.Request) {
	// Grab config from env vars
	projectID := os.Getenv("GCP_PROJECT_ID")
	location := os.Getenv("GCP_LOCATION") // e.g., us-central1
	queue := os.Getenv("CLOUD_TASKS_QUEUE")
	workerURL := os.Getenv("CLOUD_RUN_URL") + "/api/jobs/sync/worker"
	saEmail := os.Getenv("SCHEDULER_SERVICE_ACCOUNT")

	err := h.Service.TriggerGlobalSync(r.Context(), projectID, location, queue, workerURL, saEmail)
	if err != nil {
		slog.Error("failed to trigger global sync", "error", err)
		http.Error(w, "Trigger failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "tasks_enqueued"}`))
}

// 2. The Worker Handler (Called by Cloud Tasks)
func (h *Handler) handleSyncWorker(w http.ResponseWriter, r *http.Request) {
	// Parse the user ID from the Cloud Task payload
	var payload struct {
		UserID uint `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Execute the sync for just this user
	err := h.Service.SyncSingleUser(r.Context(), payload.UserID)
	if err != nil {
		slog.Error("user sync failed", "userID", payload.UserID, "error", err)
		// Returning a 500 tells Cloud Tasks to automatically retry this task later!
		http.Error(w, "Sync failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
