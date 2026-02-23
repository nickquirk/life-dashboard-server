package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func toTaskResponse(t domain.Task) domain.TaskResponse {
	resp := domain.TaskResponse{
		ID:           t.ID,
		Parent:       t.Parent,
		TaskListID:   t.TaskListID,
		Title:        t.Title,
		Status:       t.Status,
		Due:          t.Due,
		Notes:        t.Notes,
		Updated:      t.Updated,
		DurationMins: t.DurationMins,
		Date:         t.Date,
		IsRepeating:  t.IsRepeating,
		Quadrant:     t.Quadrant,
	}
	for _, s := range t.Subtasks {
		resp.Subtasks = append(resp.Subtasks, toTaskResponse(s))
	}
	return resp
}

func toTaskListResponse(tl domain.TaskList) domain.TaskListResponse {
	resp := domain.TaskListResponse{
		ID:    tl.ID,
		Title: tl.Title,
	}
	for _, t := range tl.Tasks {
		resp.Tasks = append(resp.Tasks, toTaskResponse(t))
	}
	return resp
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

	resp := make([]domain.TaskListResponse, len(lists))
	for i, l := range lists {
		resp[i] = toTaskListResponse(l)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
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
		http.Error(w, "Failed to sync task lists", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"synced"}`))
}

// POST /api/tasks/{taskListId}
func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(UserIDKey).(uint)
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

	task, err := h.Service.CreateTask(ctx, userID, taskListID, req)
	if err != nil {
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toTaskResponse(task))
}

// GET /api/tasks/{taskListId}
func (h *Handler) getTasksInList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := ctx.Value(UserIDKey).(uint)
	if !ok {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	taskListID := chi.URLParam(r, "taskListId")

	tasks, err := h.Service.GetTasks(ctx, userID, taskListID)
	if err != nil {
		http.Error(w, "Failed to fetch tasks", http.StatusInternalServerError)
		return
	}

	resp := make([]domain.TaskResponse, len(tasks))
	for i, t := range tasks {
		resp[i] = toTaskResponse(t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
		http.Error(w, "Failed to sync tasks", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "synced"}`))
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
		http.Error(w, "Failed to update task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Task updated"}`))
}

// DELETE /api/tasks/{id}
func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := chi.URLParam(r, "id")

	userID, ok := ctx.Value(UserIDKey).(uint)
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	if err := h.Service.DeleteTask(ctx, userID, taskID); err != nil {
		http.Error(w, "Failed to delete task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Task deleted"}`))
}
