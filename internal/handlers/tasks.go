package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func helloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!\n"))
}

func (h *Handler) getTaskLists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := ctx.Value(UserIDKey).(uint)
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	lists, err := h.Service.SyncAndGetTaskLists(ctx, userID)
	if err != nil {
		// In production, log the specific error but return generic to user
		http.Error(w, "Failed to retrieve task lists: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lists)
}

func (h *Handler) getTasksInList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskListID := chi.URLParam(r, "taskListId")

	userID, ok := ctx.Value(UserIDKey).(uint)
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	tasks, err := h.Service.SyncAndGetAllTasks(ctx, userID, taskListID)
	if err != nil {
		http.Error(w, "Failed to retrieve tasks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// Should this sync from google too?
func (h *Handler) getActiveTasksInList(w http.ResponseWriter, r *http.Request) {
	taskListID := chi.URLParam(r, "taskListId")

	tasks, err := h.Service.GetActiveTasks(r.Context(), taskListID)
	if err != nil {
		http.Error(w, "Failed to fetch active tasks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}
