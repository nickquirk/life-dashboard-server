package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTaskService(taskRepo *mocks.MockTaskRepository) Service {
	return NewServiceWithRepos(nil, taskRepo, nil, nil, nil, nil, nil, nil)
}

// newTaskServiceWithUserRepo creates a service with both a user and task repo.
// Needed for tests that exercise code paths reaching the Google client (which
// calls userRepo.Get to obtain tokens).
func newTaskServiceWithUserRepo(userRepo *mocks.MockUserRepository, taskRepo *mocks.MockTaskRepository) Service {
	return NewServiceWithRepos(userRepo, taskRepo, nil, nil, nil, nil, nil, nil)
}

func intPtr(v int) *int { return &v }

// =====================================================================
// GetTaskLists
// =====================================================================

func TestService_GetTaskLists_Success(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		GetTaskListsFunc: func(userID uint) ([]domain.TaskList, error) {
			return []domain.TaskList{{ID: "list1", Title: "My Tasks"}}, nil
		},
	}
	svc := newTaskService(repo)

	resp, err := svc.GetTaskLists(domain.GetTaskListsRequest{UserID: 1})
	require.NoError(t, err)
	assert.Len(t, resp.TaskLists, 1)
	assert.Equal(t, "My Tasks", resp.TaskLists[0].Title)
}

func TestService_GetTaskLists_Empty(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		GetTaskListsFunc: func(userID uint) ([]domain.TaskList, error) {
			return []domain.TaskList{}, nil
		},
	}
	svc := newTaskService(repo)

	resp, err := svc.GetTaskLists(domain.GetTaskListsRequest{UserID: 1})
	require.NoError(t, err)
	assert.Empty(t, resp.TaskLists)
}

func TestService_GetTaskLists_Error(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		GetTaskListsFunc: func(userID uint) ([]domain.TaskList, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTaskService(repo)

	_, err := svc.GetTaskLists(domain.GetTaskListsRequest{UserID: 1})
	assert.Error(t, err)
}

func TestService_GetTaskLists_PassesCorrectUserID(t *testing.T) {
	var capturedUserID uint
	repo := &mocks.MockTaskRepository{
		GetTaskListsFunc: func(userID uint) ([]domain.TaskList, error) {
			capturedUserID = userID
			return []domain.TaskList{}, nil
		},
	}
	svc := newTaskService(repo)

	_, err := svc.GetTaskLists(domain.GetTaskListsRequest{UserID: 42})
	require.NoError(t, err)
	assert.Equal(t, uint(42), capturedUserID)
}

// =====================================================================
// GetTasks
// =====================================================================

func TestService_GetTasks_Success(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return nil
		},
		GetTasksFunc: func(taskListID string) ([]domain.Task, error) {
			return []domain.Task{{ID: "t1", Title: "Task 1"}}, nil
		},
	}
	svc := newTaskService(repo)

	resp, err := svc.GetTasks(context.Background(), domain.GetTasksRequest{UserID: 1, TaskListID: "list1"})
	require.NoError(t, err)
	assert.Len(t, resp.Tasks, 1)
	assert.Equal(t, "Task 1", resp.Tasks[0].Title)
}

func TestService_GetTasks_Empty(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return nil
		},
		GetTasksFunc: func(taskListID string) ([]domain.Task, error) {
			return []domain.Task{}, nil
		},
	}
	svc := newTaskService(repo)

	resp, err := svc.GetTasks(context.Background(), domain.GetTasksRequest{UserID: 1, TaskListID: "list1"})
	require.NoError(t, err)
	assert.Empty(t, resp.Tasks)
}

func TestService_GetTasks_OwnershipFailure(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return errors.New("record not found")
		},
		GetTasksFunc: func(taskListID string) ([]domain.Task, error) {
			t.Fatal("GetTasks should not be called when ownership check fails")
			return nil, nil
		},
	}
	svc := newTaskService(repo)

	_, err := svc.GetTasks(context.Background(), domain.GetTasksRequest{UserID: 1, TaskListID: "not-my-list"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task list not found")
}

func TestService_GetTasks_RepoError(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return nil
		},
		GetTasksFunc: func(taskListID string) ([]domain.Task, error) {
			return nil, errors.New("db timeout")
		},
	}
	svc := newTaskService(repo)

	_, err := svc.GetTasks(context.Background(), domain.GetTasksRequest{UserID: 1, TaskListID: "list1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db timeout")
}

func TestService_GetTasks_PassesCorrectTaskListID(t *testing.T) {
	var capturedListID string
	repo := &mocks.MockTaskRepository{
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return nil
		},
		GetTasksFunc: func(taskListID string) ([]domain.Task, error) {
			capturedListID = taskListID
			return []domain.Task{}, nil
		},
	}
	svc := newTaskService(repo)

	_, err := svc.GetTasks(context.Background(), domain.GetTasksRequest{UserID: 1, TaskListID: "specific-list"})
	require.NoError(t, err)
	assert.Equal(t, "specific-list", capturedListID)
}

// =====================================================================
// UpdateTask (local-only fields — no Google API call when shouldPatch=false)
// =====================================================================

func TestService_UpdateTask_GetTaskListIDError(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "", errors.New("task not in db")
		},
	}
	svc := newTaskService(repo)

	_, err := svc.UpdateTask(context.Background(), domain.UpdateTaskRequest{
		UserID: 1, TaskID: "unknown",
	})
	assert.Error(t, err)
}

func TestService_UpdateTask_OwnershipError(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "list1", nil
		},
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return errors.New("not owner")
		},
	}
	svc := newTaskService(repo)

	_, err := svc.UpdateTask(context.Background(), domain.UpdateTaskRequest{
		UserID: 1, TaskID: "t1",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestService_UpdateTask_LocalOnly_Quadrant(t *testing.T) {
	var capturedUpdates map[string]any
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "list1", nil
		},
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return nil
		},
		UpdateTaskFunc: func(taskID string, updates map[string]any) error {
			capturedUpdates = updates
			return nil
		},
	}
	svc := newTaskService(repo)

	resp, err := svc.UpdateTask(context.Background(), domain.UpdateTaskRequest{
		UserID: 1, TaskID: "t1", Quadrant: intPtr(2),
	})
	require.NoError(t, err)
	assert.Equal(t, "t1", resp.ID)
	assert.Equal(t, 2, capturedUpdates["quadrant"])
	assert.NotContains(t, capturedUpdates, "title")
}

func TestService_UpdateTask_LocalOnly_DurationMins(t *testing.T) {
	var capturedUpdates map[string]any
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "list1", nil
		},
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return nil
		},
		UpdateTaskFunc: func(taskID string, updates map[string]any) error {
			capturedUpdates = updates
			return nil
		},
	}
	svc := newTaskService(repo)

	resp, err := svc.UpdateTask(context.Background(), domain.UpdateTaskRequest{
		UserID: 1, TaskID: "t1", DurationMins: intPtr(30),
	})
	require.NoError(t, err)
	assert.Equal(t, "t1", resp.ID)
	assert.Equal(t, 30, capturedUpdates["duration_mins"])
}

func TestService_UpdateTask_LocalOnly_SetDate(t *testing.T) {
	var capturedUpdates map[string]any
	date := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "list1", nil
		},
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return nil
		},
		UpdateTaskFunc: func(taskID string, updates map[string]any) error {
			capturedUpdates = updates
			return nil
		},
	}
	svc := newTaskService(repo)

	resp, err := svc.UpdateTask(context.Background(), domain.UpdateTaskRequest{
		UserID: 1, TaskID: "t1",
		Date: &domain.NullableDate{Time: date, Valid: true},
	})
	require.NoError(t, err)
	assert.Equal(t, "t1", resp.ID)
	assert.Equal(t, date, capturedUpdates["date"])
}

func TestService_UpdateTask_LocalOnly_ClearDate(t *testing.T) {
	var capturedUpdates map[string]any
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "list1", nil
		},
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return nil
		},
		UpdateTaskFunc: func(taskID string, updates map[string]any) error {
			capturedUpdates = updates
			return nil
		},
	}
	svc := newTaskService(repo)

	resp, err := svc.UpdateTask(context.Background(), domain.UpdateTaskRequest{
		UserID: 1, TaskID: "t1",
		Date: &domain.NullableDate{Valid: false},
	})
	require.NoError(t, err)
	assert.Equal(t, "t1", resp.ID)
	assert.Nil(t, capturedUpdates["date"])
}

func TestService_UpdateTask_LocalOnly_MultipleFields(t *testing.T) {
	var capturedUpdates map[string]any
	date := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "list1", nil
		},
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return nil
		},
		UpdateTaskFunc: func(taskID string, updates map[string]any) error {
			capturedUpdates = updates
			return nil
		},
	}
	svc := newTaskService(repo)

	resp, err := svc.UpdateTask(context.Background(), domain.UpdateTaskRequest{
		UserID: 1, TaskID: "t1",
		Quadrant:     intPtr(3),
		DurationMins: intPtr(45),
		Date:         &domain.NullableDate{Time: date, Valid: true},
	})
	require.NoError(t, err)
	assert.Equal(t, "t1", resp.ID)
	assert.Equal(t, 3, capturedUpdates["quadrant"])
	assert.Equal(t, 45, capturedUpdates["duration_mins"])
	assert.Equal(t, date, capturedUpdates["date"])
}

func TestService_UpdateTask_LocalOnly_NoFieldsSet(t *testing.T) {
	var updateCalled bool
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "list1", nil
		},
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return nil
		},
		UpdateTaskFunc: func(taskID string, updates map[string]any) error {
			updateCalled = true
			assert.Empty(t, updates)
			return nil
		},
	}
	svc := newTaskService(repo)

	resp, err := svc.UpdateTask(context.Background(), domain.UpdateTaskRequest{
		UserID: 1, TaskID: "t1",
	})
	require.NoError(t, err)
	assert.Equal(t, "t1", resp.ID)
	assert.True(t, updateCalled)
}

func TestService_UpdateTask_LocalOnly_RepoError(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "list1", nil
		},
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return nil
		},
		UpdateTaskFunc: func(taskID string, updates map[string]any) error {
			return errors.New("constraint violation")
		},
	}
	svc := newTaskService(repo)

	_, err := svc.UpdateTask(context.Background(), domain.UpdateTaskRequest{
		UserID: 1, TaskID: "t1", Quadrant: intPtr(1),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "constraint violation")
}

// =====================================================================
// DeleteTask (pre-Google validation paths)
// =====================================================================

func TestService_DeleteTask_TaskNotFound(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "", errors.New("record not found")
		},
	}
	svc := newTaskService(repo)

	_, err := svc.DeleteTask(context.Background(), domain.DeleteTaskRequest{
		UserID: 1, TaskID: "nonexistent",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestService_DeleteTask_OwnershipError(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "list1", nil
		},
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return errors.New("not owner")
		},
	}
	svc := newTaskService(repo)

	_, err := svc.DeleteTask(context.Background(), domain.DeleteTaskRequest{
		UserID: 1, TaskID: "t1",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestService_DeleteTask_GetTasksError(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "list1", nil
		},
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return nil
		},
		GetTasksFunc: func(taskListID string) ([]domain.Task, error) {
			return nil, errors.New("db error fetching tasks")
		},
	}
	svc := newTaskService(repo)

	_, err := svc.DeleteTask(context.Background(), domain.DeleteTaskRequest{
		UserID: 1, TaskID: "t1",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch tasks for list")
}

func TestService_DeleteTask_CollectsSubtasks(t *testing.T) {
	parentID := "parent1"
	userRepo := &mocks.MockUserRepository{
		GetFunc: func(req domain.GetUserRequest) (domain.GetUserResponse, error) {
			return domain.GetUserResponse{}, errors.New("no tokens")
		},
	}
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "list1", nil
		},
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return nil
		},
		GetTasksFunc: func(taskListID string) ([]domain.Task, error) {
			return []domain.Task{
				{ID: "parent1", Title: "Parent"},
				{ID: "child1", Parent: &parentID, Title: "Child 1"},
				{ID: "child2", Parent: &parentID, Title: "Child 2"},
				{ID: "unrelated", Title: "Unrelated Task"},
			}, nil
		},
	}
	svc := newTaskServiceWithUserRepo(userRepo, repo)

	// Subtask collection runs, then fails at Google client (user has no tokens)
	_, err := svc.DeleteTask(context.Background(), domain.DeleteTaskRequest{
		UserID: 1, TaskID: "parent1",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Google client")
}

func TestService_DeleteTask_NoSubtasks(t *testing.T) {
	userRepo := &mocks.MockUserRepository{
		GetFunc: func(req domain.GetUserRequest) (domain.GetUserResponse, error) {
			return domain.GetUserResponse{}, errors.New("no tokens")
		},
	}
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "list1", nil
		},
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return nil
		},
		GetTasksFunc: func(taskListID string) ([]domain.Task, error) {
			return []domain.Task{
				{ID: "t1", Title: "Solo Task"},
				{ID: "t2", Title: "Other Task"},
			}, nil
		},
	}
	svc := newTaskServiceWithUserRepo(userRepo, repo)

	_, err := svc.DeleteTask(context.Background(), domain.DeleteTaskRequest{
		UserID: 1, TaskID: "t1",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Google client")
}

// =====================================================================
// DeleteTasks (batch)
// =====================================================================

func TestService_DeleteTasks_EmptyIDs(t *testing.T) {
	repo := &mocks.MockTaskRepository{}
	svc := newTaskService(repo)

	resp, err := svc.DeleteTasks(context.Background(), domain.DeleteTasksRequest{
		UserID: 1, TaskIDs: []string{},
	})
	require.NoError(t, err)
	assert.Empty(t, resp.IDs)
}

func TestService_DeleteTasks_TaskNotFound(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "", errors.New("record not found")
		},
	}
	svc := newTaskService(repo)

	_, err := svc.DeleteTasks(context.Background(), domain.DeleteTasksRequest{
		UserID: 1, TaskIDs: []string{"nonexistent"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestService_DeleteTasks_OwnershipError(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			return "list1", nil
		},
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			return errors.New("not owner")
		},
		GetTasksFunc: func(taskListID string) ([]domain.Task, error) {
			return []domain.Task{}, nil
		},
	}
	svc := newTaskService(repo)

	_, err := svc.DeleteTasks(context.Background(), domain.DeleteTasksRequest{
		UserID: 1, TaskIDs: []string{"t1"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestService_DeleteTasks_SecondTaskNotFound(t *testing.T) {
	callCount := 0
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			callCount++
			if taskID == "t2" {
				return "", errors.New("record not found")
			}
			return "list1", nil
		},
	}
	svc := newTaskService(repo)

	_, err := svc.DeleteTasks(context.Background(), domain.DeleteTasksRequest{
		UserID: 1, TaskIDs: []string{"t1", "t2"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found: t2")
}

func TestService_DeleteTasks_GroupsTasksByList(t *testing.T) {
	// Tasks from two different lists — verify both lists get ownership checked
	var verifiedLists []string
	userRepo := &mocks.MockUserRepository{
		GetFunc: func(req domain.GetUserRequest) (domain.GetUserResponse, error) {
			return domain.GetUserResponse{}, errors.New("no tokens")
		},
	}
	repo := &mocks.MockTaskRepository{
		GetTaskListIDForTaskFunc: func(taskID string) (string, error) {
			if taskID == "t1" {
				return "listA", nil
			}
			return "listB", nil
		},
		VerifyTaskListOwnerFunc: func(userID uint, taskListID string) error {
			verifiedLists = append(verifiedLists, taskListID)
			return nil
		},
		GetTasksFunc: func(taskListID string) ([]domain.Task, error) {
			return []domain.Task{}, nil
		},
	}
	svc := newTaskServiceWithUserRepo(userRepo, repo)

	// Will fail at Google client, but ownership verification ran
	_, err := svc.DeleteTasks(context.Background(), domain.DeleteTasksRequest{
		UserID: 1, TaskIDs: []string{"t1", "t2"},
	})
	assert.Error(t, err) // Fails at Google client
	assert.Contains(t, verifiedLists, "listA")
	assert.Contains(t, verifiedLists, "listB")
}
