package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func helloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!\n"))
}

// GET /api/tasks
func (h *Handler) getTaskLists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(UserIDKey).(uint)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// FAST READ
	lists, err := h.Service.GetTaskLists(userID)
	if err != nil {
		http.Error(w, "Failed to fetch lists", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lists)
}

// POST /api/tasks/sync
func (h *Handler) syncTaskLists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(UserIDKey).(uint)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// SLOW SYNC
	if err := h.Service.SyncTaskLists(ctx, userID); err != nil {
		http.Error(w, "Sync failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"synced"}`))
}

// POST /api/tasks/{taskListId}
func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskListID := chi.URLParam(r, "taskListId")
	userID := ctx.Value(UserIDKey).(uint)

	var req domain.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	task, err := h.Service.CreateTask(ctx, userID, taskListID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// GET /api/tasks/{taskListId}
func (h *Handler) getTasksInList(w http.ResponseWriter, r *http.Request) {
	taskListID := chi.URLParam(r, "taskListId")

	// Just read from DB
	tasks, err := h.Service.GetTasks(r.Context(), taskListID)
	if err != nil {
		http.Error(w, "Failed to fetch tasks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// POST /api/tasks/{taskListId}/sync - New Endpoint
func (h *Handler) syncTasksInList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskListID := chi.URLParam(r, "taskListId")

	userID, ok := ctx.Value(UserIDKey).(uint)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Perform the heavy lifting
	err := h.Service.SyncTasks(ctx, userID, taskListID)
	if err != nil {
		// Log error in production
		http.Error(w, "Sync failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "synced"}`))
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

func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := chi.URLParam(r, "id")

	userID, ok := ctx.Value(UserIDKey).(uint)
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	var req domain.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.Service.UpdateTask(ctx, userID, taskID, req); err != nil {
		http.Error(w, "Failed to update task: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Task updated"}`))
}
