package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

// GET /api/tasks
func (h *Handler) getTaskLists(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	resp, err := h.Service.GetTaskLists(domain.GetTaskListsRequest{UserID: userID})
	if err != nil {
		http.Error(w, "Failed to fetch lists", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp.TaskLists); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// POST /api/tasks/sync
func (h *Handler) syncTaskLists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	resp, err := h.Service.SyncTaskLists(ctx, domain.SyncTaskListsRequest{UserID: userID})
	if err != nil {
		http.Error(w, "Failed to sync task lists", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// POST /api/tasks/{taskListId}
func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	taskListID := chi.URLParam(r, "taskListId")

	var req domain.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	req.UserID = userID
	req.TaskListID = taskListID

	resp, err := h.Service.CreateTask(ctx, req)
	if err != nil {
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp.TaskResponse)
}

// GET /api/tasks/{taskListId}
func (h *Handler) getTasksInList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	taskListID := chi.URLParam(r, "taskListId")

	resp, err := h.Service.GetTasks(ctx, domain.GetTasksRequest{UserID: userID, TaskListID: taskListID})
	if err != nil {
		http.Error(w, "Failed to fetch tasks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp.Tasks)
}

// POST /api/tasks/{taskListId}/sync - New Endpoint
func (h *Handler) syncTasksInList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskListID := chi.URLParam(r, "taskListId")

	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	resp, err := h.Service.SyncTasks(ctx, domain.SyncTasksRequest{UserID: userID, TaskListID: taskListID})
	if err != nil {
		http.Error(w, "Failed to sync tasks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := chi.URLParam(r, "id")

	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	var req domain.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.UserID = userID
	req.TaskID = taskID

	resp, err := h.Service.UpdateTask(ctx, req)
	if err != nil {
		http.Error(w, "Failed to update task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// DELETE /api/tasks/{id}
func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := chi.URLParam(r, "id")

	userID, ok := h.GetUserID(r)
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	resp, err := h.Service.DeleteTask(ctx, domain.DeleteTaskRequest{UserID: userID, TaskID: taskID})
	if err != nil {
		http.Error(w, "Failed to delete task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
