package service

import (
	"context"
	"errors"
	"testing"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTaskService(taskRepo *mocks.MockTaskRepository) Service {
	return NewServiceWithRepos(nil, taskRepo, nil, nil, nil)
}

// --- GetTaskLists ---

func TestService_GetTaskLists_Success(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		GetTaskListsFunc: func(userID uint) ([]domain.TaskList, error) {
			return []domain.TaskList{{ID: "list1", Title: "My Tasks"}}, nil
		},
	}
	svc := newTaskService(repo)

	lists, err := svc.GetTaskLists(1)
	require.NoError(t, err)
	assert.Len(t, lists, 1)
	assert.Equal(t, "My Tasks", lists[0].Title)
}

func TestService_GetTaskLists_Error(t *testing.T) {
	repo := &mocks.MockTaskRepository{
		GetTaskListsFunc: func(userID uint) ([]domain.TaskList, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTaskService(repo)

	_, err := svc.GetTaskLists(1)
	assert.Error(t, err)
}

// --- GetTasks ---

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

	tasks, err := svc.GetTasks(context.Background(), 1, "list1")
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "Task 1", tasks[0].Title)
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

	_, err := svc.GetTasks(context.Background(), 1, "not-my-list")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task list not found")
}
