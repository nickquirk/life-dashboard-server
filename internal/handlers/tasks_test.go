package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withUser(r *http.Request, userID uint) *http.Request {
	ctx := context.WithValue(r.Context(), UserIDKey, userID)
	return r.WithContext(ctx)
}

func withUserAndChi(r *http.Request, userID uint, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, UserIDKey, userID)
	return r.WithContext(ctx)
}

// --- getTaskLists ---

func TestGetTaskLists_Success(t *testing.T) {
	svc := &mocks.MockService{
		GetTaskListsFunc: func(req domain.GetTaskListsRequest) (domain.GetTaskListsResponse, error) {
			return domain.GetTaskListsResponse{
				TaskLists: []domain.TaskList{
					{ID: "list1", Title: "My Tasks", Tasks: []domain.Task{
						{ID: "t1", Title: "Do laundry", Status: "needsAction"},
					}},
				},
			}, nil
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.getTaskLists(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "My Tasks")
	assert.Contains(t, rr.Body.String(), "Do laundry")
}

func TestGetTaskLists_NoUser(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rr := httptest.NewRecorder()

	h.getTaskLists(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestGetTaskLists_ServiceError(t *testing.T) {
	svc := &mocks.MockService{
		GetTaskListsFunc: func(req domain.GetTaskListsRequest) (domain.GetTaskListsResponse, error) {
			return domain.GetTaskListsResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.getTaskLists(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- syncTaskLists ---

func TestSyncTaskLists_Success(t *testing.T) {
	svc := &mocks.MockService{
		SyncTaskListsFunc: func(ctx context.Context, req domain.SyncTaskListsRequest) (domain.SyncTaskListsResponse, error) {
			return domain.SyncTaskListsResponse{
				TaskLists: []domain.TaskList{{ID: "list1", Title: "Synced List"}},
			}, nil
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodPost, "/api/tasks/sync", nil)
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.syncTaskLists(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Synced List")
}

func TestSyncTaskLists_Error(t *testing.T) {
	svc := &mocks.MockService{
		SyncTaskListsFunc: func(ctx context.Context, req domain.SyncTaskListsRequest) (domain.SyncTaskListsResponse, error) {
			return domain.SyncTaskListsResponse{}, errors.New("sync failed")
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodPost, "/api/tasks/sync", nil)
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.syncTaskLists(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- createTask ---

func TestCreateTask_Success(t *testing.T) {
	svc := &mocks.MockService{
		CreateTaskFunc: func(ctx context.Context, req domain.CreateTaskRequest) (domain.CreateTaskResponse, error) {
			return domain.CreateTaskResponse{
				ID: "new-task", Title: req.Title, Status: "needsAction",
			}, nil
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.CreateTaskRequest{Title: "Buy milk"})
	r := httptest.NewRequest(http.MethodPost, "/api/tasks/list1", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "taskListId", "list1")
	rr := httptest.NewRecorder()

	h.createTask(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Buy milk")
}

func TestCreateTask_BadBody(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPost, "/api/tasks/list1", bytes.NewReader([]byte("invalid")))
	r = withUserAndChi(r, 1, "taskListId", "list1")
	rr := httptest.NewRecorder()

	h.createTask(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateTask_ServiceError(t *testing.T) {
	svc := &mocks.MockService{
		CreateTaskFunc: func(ctx context.Context, req domain.CreateTaskRequest) (domain.CreateTaskResponse, error) {
			return domain.CreateTaskResponse{}, errors.New("google error")
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.CreateTaskRequest{Title: "Buy milk"})
	r := httptest.NewRequest(http.MethodPost, "/api/tasks/list1", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "taskListId", "list1")
	rr := httptest.NewRecorder()

	h.createTask(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- getTasksInList ---

func TestGetTasksInList_Success(t *testing.T) {
	svc := &mocks.MockService{
		GetTasksFunc: func(ctx context.Context, req domain.GetTasksRequest) (domain.GetTasksResponse, error) {
			return domain.GetTasksResponse{
				Tasks: []domain.Task{{ID: "t1", Title: "Task 1", Status: "needsAction"}},
			}, nil
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/api/tasks/list1", nil)
	r = withUserAndChi(r, 1, "taskListId", "list1")
	rr := httptest.NewRecorder()

	h.getTasksInList(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Task 1")
}

func TestGetTasksInList_Error(t *testing.T) {
	svc := &mocks.MockService{
		GetTasksFunc: func(ctx context.Context, req domain.GetTasksRequest) (domain.GetTasksResponse, error) {
			return domain.GetTasksResponse{}, errors.New("error")
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/api/tasks/list1", nil)
	r = withUserAndChi(r, 1, "taskListId", "list1")
	rr := httptest.NewRecorder()

	h.getTasksInList(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- syncTasksInList ---

func TestSyncTasksInList_Success(t *testing.T) {
	svc := &mocks.MockService{
		SyncTasksFunc: func(ctx context.Context, req domain.SyncTasksRequest) (domain.SyncTasksResponse, error) {
			return domain.SyncTasksResponse{
				Tasks: []domain.Task{{ID: "t1", Title: "Synced Task"}},
			}, nil
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodPost, "/api/tasks/list1/sync", nil)
	r = withUserAndChi(r, 1, "taskListId", "list1")
	rr := httptest.NewRecorder()

	h.syncTasksInList(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Synced Task")
}

func TestSyncTasksInList_Error(t *testing.T) {
	svc := &mocks.MockService{
		SyncTasksFunc: func(ctx context.Context, req domain.SyncTasksRequest) (domain.SyncTasksResponse, error) {
			return domain.SyncTasksResponse{}, errors.New("sync failed")
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodPost, "/api/tasks/list1/sync", nil)
	r = withUserAndChi(r, 1, "taskListId", "list1")
	rr := httptest.NewRecorder()

	h.syncTasksInList(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- updateTask ---

func TestUpdateTask_Success(t *testing.T) {
	title := "Updated Title"
	svc := &mocks.MockService{
		UpdateTaskFunc: func(ctx context.Context, req domain.UpdateTaskRequest) (domain.UpdateTaskResponse, error) {
			require.Equal(t, "task1", req.TaskID)
			require.Equal(t, &title, req.Title)
			return domain.UpdateTaskResponse{ID: "task1"}, nil
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.UpdateTaskRequest{Title: &title})
	r := httptest.NewRequest(http.MethodPatch, "/api/tasks/task1", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "task1")
	rr := httptest.NewRecorder()

	h.updateTask(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "task1")
}

func TestUpdateTask_BadBody(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPatch, "/api/tasks/task1", bytes.NewReader([]byte("bad")))
	r = withUserAndChi(r, 1, "id", "task1")
	rr := httptest.NewRecorder()

	h.updateTask(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateTask_ServiceError(t *testing.T) {
	title := "Updated"
	svc := &mocks.MockService{
		UpdateTaskFunc: func(ctx context.Context, req domain.UpdateTaskRequest) (domain.UpdateTaskResponse, error) {
			return domain.UpdateTaskResponse{}, errors.New("update failed")
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.UpdateTaskRequest{Title: &title})
	r := httptest.NewRequest(http.MethodPatch, "/api/tasks/task1", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "task1")
	rr := httptest.NewRecorder()

	h.updateTask(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- deleteTask ---

func TestDeleteTask_Success(t *testing.T) {
	svc := &mocks.MockService{
		DeleteTaskFunc: func(ctx context.Context, req domain.DeleteTaskRequest) (domain.DeleteTaskResponse, error) {
			return domain.DeleteTaskResponse{ID: "task1"}, nil
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodDelete, "/api/tasks/task1", nil)
	r = withUserAndChi(r, 1, "id", "task1")
	rr := httptest.NewRecorder()

	h.deleteTask(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "task1")
}

func TestDeleteTask_Error(t *testing.T) {
	svc := &mocks.MockService{
		DeleteTaskFunc: func(ctx context.Context, req domain.DeleteTaskRequest) (domain.DeleteTaskResponse, error) {
			return domain.DeleteTaskResponse{}, errors.New("delete failed")
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodDelete, "/api/tasks/task1", nil)
	r = withUserAndChi(r, 1, "id", "task1")
	rr := httptest.NewRecorder()

	h.deleteTask(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
