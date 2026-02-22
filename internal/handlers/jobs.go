package handlers

import (
	"log"
	"net/http"
	"os"
)

func (h *Handler) handleGlobalSync(w http.ResponseWriter, r *http.Request) {
	// Security Check (Simple Secret)
	// Cloud Scheduler will send this header
	if r.Header.Get("X-Scheduler-Secret") != os.Getenv("SCHEDULER_SECRET") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Execute the Job
	// Cloud Run can kill requests after 60 mins, but usually keep this shorter
	err := h.Service.SyncAllUsers(r.Context())
	if err != nil {
		log.Printf("Global sync failed: %v", err)
		http.Error(w, "Sync failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Sync complete"))
}
