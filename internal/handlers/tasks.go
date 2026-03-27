package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

// GET /api/tasks
func (h *Handler) getTaskLists(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	resp, err := h.Service.GetTaskLists(domain.GetTaskListsRequest{UserID: userID})
	if err != nil {
		h.respondWithError(w, "Failed to fetch lists", err, http.StatusInternalServerError, "userID", userID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.respondWithError(w, "Failed to encode response", err, http.StatusInternalServerError)
		return
	}
}

// POST /api/tasks/sync
func (h *Handler) syncTaskLists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	resp, err := h.Service.SyncTaskLists(ctx, domain.SyncTaskListsRequest{UserID: userID})
	if err != nil {
		h.respondWithError(w, "Failed to sync task lists", err, http.StatusInternalServerError, "userID", userID)
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
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	taskListID := chi.URLParam(r, "taskListId")

	var req domain.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.UserID = userID
	req.TaskListID = taskListID

	resp, err := h.Service.CreateTask(ctx, req)
	if err != nil {
		h.respondWithError(w, "Failed to create task", err, http.StatusInternalServerError, "userID", userID, "taskListID", taskListID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GET /api/tasks/{taskListId}
func (h *Handler) getTasksInList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	taskListID := chi.URLParam(r, "taskListId")

	resp, err := h.Service.GetTasks(ctx, domain.GetTasksRequest{UserID: userID, TaskListID: taskListID})
	if err != nil {
		h.respondWithError(w, "Failed to fetch tasks", err, http.StatusInternalServerError, "userID", userID, "taskListID", taskListID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// POST /api/tasks/{taskListId}/sync - New Endpoint
func (h *Handler) syncTasksInList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskListID := chi.URLParam(r, "taskListId")

	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	resp, err := h.Service.SyncTasks(ctx, domain.SyncTasksRequest{UserID: userID, TaskListID: taskListID})
	if err != nil {
		h.respondWithError(w, "Failed to sync tasks", err, http.StatusInternalServerError, "userID", userID, "taskListID", taskListID)
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
		h.respondWithError(w, "User not found in context", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	var req domain.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID, "taskID", taskID)
		return
	}
	req.UserID = userID
	req.TaskID = taskID

	resp, err := h.Service.UpdateTask(ctx, req)
	if err != nil {
		h.respondWithError(w, "Failed to update task", err, http.StatusInternalServerError, "userID", userID, "taskID", taskID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// DELETE /api/tasks
func (h *Handler) deleteTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := h.GetUserID(r)
	if !ok {
		h.respondWithError(w, "User not found in context", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	var req domain.DeleteTasksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, "Invalid request body", err, http.StatusBadRequest, "userID", userID)
		return
	}
	req.UserID = userID

	resp, err := h.Service.DeleteTasks(ctx, req)
	if err != nil {
		h.respondWithError(w, "Failed to delete tasks", err, http.StatusInternalServerError, "userID", userID)
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
		h.respondWithError(w, "User not found in context", fmt.Errorf("user ID not in context"), http.StatusUnauthorized)
		return
	}

	resp, err := h.Service.DeleteTask(ctx, domain.DeleteTaskRequest{UserID: userID, TaskID: taskID})
	if err != nil {
		h.respondWithError(w, "Failed to delete task", err, http.StatusInternalServerError, "userID", userID, "taskID", taskID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
