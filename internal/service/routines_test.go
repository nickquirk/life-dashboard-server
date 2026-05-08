package service

import (
	"errors"
	"testing"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRoutineService(repo *mocks.MockRoutineRepository) Service {
	return NewServiceWithRepos(nil, nil, nil, nil, repo, nil, nil, nil)
}

func ptrInt(v int) *int       { return &v }
func ptrStr(v string) *string { return &v }

// --- CreateRoutine ---

func TestService_CreateRoutine_Success_PassesAllFields(t *testing.T) {
	var captured domain.Routine
	repo := &mocks.MockRoutineRepository{
		CreateRoutineFunc: func(r domain.Routine) (domain.Routine, error) {
			captured = r
			r.ID = 7
			return r, nil
		},
	}
	svc := newRoutineService(repo)

	req := domain.CreateRoutineRequest{
		UserID:       1,
		Title:        "Read",
		DurationMins: 15,
		Goal:         &domain.Goal{Type: domain.GoalTypeTime, Target: 60},
		ResetPeriod:  ptrStr(domain.ResetPeriodWeekly),
	}
	resp, err := svc.CreateRoutine(req)
	require.NoError(t, err)

	assert.Equal(t, uint(7), resp.ID)
	assert.Equal(t, "Read", resp.Title)
	assert.Equal(t, 15, resp.DurationMins)
	require.NotNil(t, resp.Goal)
	assert.Equal(t, domain.GoalTypeTime, resp.Goal.Type)
	assert.Equal(t, 60, resp.Goal.Target)
	require.NotNil(t, resp.ResetPeriod)
	assert.Equal(t, domain.ResetPeriodWeekly, *resp.ResetPeriod)

	assert.Equal(t, uint(1), captured.UserID)
	assert.Equal(t, "Read", captured.Title)
	require.NotNil(t, captured.GoalType)
	assert.Equal(t, "time", *captured.GoalType)
	require.NotNil(t, captured.GoalTarget)
	assert.Equal(t, 60, *captured.GoalTarget)
	require.NotNil(t, captured.Goal)
	assert.Equal(t, domain.GoalTypeTime, captured.Goal.Type)
	assert.Equal(t, 60, captured.Goal.Target)
}

func TestService_CreateRoutine_NilGoal_DoesNotPersistGoal(t *testing.T) {
	var captured domain.Routine
	repo := &mocks.MockRoutineRepository{
		CreateRoutineFunc: func(r domain.Routine) (domain.Routine, error) {
			captured = r
			return r, nil
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.CreateRoutine(domain.CreateRoutineRequest{
		UserID: 1, Title: "Read", DurationMins: 15,
	})
	require.NoError(t, err)
	assert.Nil(t, captured.GoalType)
	assert.Nil(t, captured.GoalTarget)
	assert.Nil(t, captured.Goal)
}

func TestService_CreateRoutine_ZeroTargetGoal_DoesNotPersistGoal(t *testing.T) {
	// Target == 0 is the "no goal / clear" sentinel — create should ignore it.
	var captured domain.Routine
	repo := &mocks.MockRoutineRepository{
		CreateRoutineFunc: func(r domain.Routine) (domain.Routine, error) {
			captured = r
			return r, nil
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.CreateRoutine(domain.CreateRoutineRequest{
		UserID: 1, Title: "Read", DurationMins: 15,
		Goal: &domain.Goal{Type: domain.GoalTypeTime, Target: 0},
	})
	require.NoError(t, err)
	assert.Nil(t, captured.GoalType)
	assert.Nil(t, captured.GoalTarget)
	assert.Nil(t, captured.Goal)
}

func TestService_CreateRoutine_CountGoal(t *testing.T) {
	var captured domain.Routine
	repo := &mocks.MockRoutineRepository{
		CreateRoutineFunc: func(r domain.Routine) (domain.Routine, error) {
			captured = r
			return r, nil
		},
	}
	svc := newRoutineService(repo)

	resp, err := svc.CreateRoutine(domain.CreateRoutineRequest{
		UserID: 1, Title: "Pushups", DurationMins: 5,
		Goal: &domain.Goal{Type: domain.GoalTypeCount, Target: 100},
	})
	require.NoError(t, err)
	require.NotNil(t, captured.GoalType)
	assert.Equal(t, "count", *captured.GoalType)
	require.NotNil(t, captured.GoalTarget)
	assert.Equal(t, 100, *captured.GoalTarget)
	require.NotNil(t, resp.Goal)
	assert.Equal(t, domain.GoalTypeCount, resp.Goal.Type)
	assert.Equal(t, 100, resp.Goal.Target)
}

func TestService_CreateRoutine_ValidationError_DoesNotCallRepo(t *testing.T) {
	called := false
	repo := &mocks.MockRoutineRepository{
		CreateRoutineFunc: func(r domain.Routine) (domain.Routine, error) {
			called = true
			return r, nil
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.CreateRoutine(domain.CreateRoutineRequest{
		UserID: 1, Title: "", DurationMins: 15,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
	assert.False(t, called, "repo should not be called when validation fails")
}

func TestService_CreateRoutine_TrimsTitleBeforeRepoCall(t *testing.T) {
	var captured domain.Routine
	repo := &mocks.MockRoutineRepository{
		CreateRoutineFunc: func(r domain.Routine) (domain.Routine, error) {
			captured = r
			return r, nil
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.CreateRoutine(domain.CreateRoutineRequest{
		UserID: 1, Title: "  Read  ", DurationMins: 15,
	})
	require.NoError(t, err)
	assert.Equal(t, "Read", captured.Title)
}

func TestService_CreateRoutine_RepoError(t *testing.T) {
	repo := &mocks.MockRoutineRepository{
		CreateRoutineFunc: func(r domain.Routine) (domain.Routine, error) {
			return domain.Routine{}, errors.New("db error")
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.CreateRoutine(domain.CreateRoutineRequest{
		UserID: 1, Title: "Read", DurationMins: 15,
	})
	assert.Error(t, err)
}

// --- UpdateRoutine ---

func TestService_UpdateRoutine_ValidationError(t *testing.T) {
	called := false
	repo := &mocks.MockRoutineRepository{
		UpdateRoutineFunc: func(userID, routineID uint, updates map[string]interface{}) error {
			called = true
			return nil
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.UpdateRoutine(domain.UpdateRoutineRequest{
		ID: 1, UserID: 1, DurationMins: ptrInt(0),
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
	assert.False(t, called)
}

func TestService_UpdateRoutine_NoFields_NoRepoCall(t *testing.T) {
	called := false
	repo := &mocks.MockRoutineRepository{
		UpdateRoutineFunc: func(userID, routineID uint, updates map[string]interface{}) error {
			called = true
			return nil
		},
	}
	svc := newRoutineService(repo)

	resp, err := svc.UpdateRoutine(domain.UpdateRoutineRequest{ID: 1, UserID: 1})
	require.NoError(t, err)
	assert.Equal(t, domain.UpdateRoutineResponse{}, resp)
	assert.False(t, called)
}

func TestService_UpdateRoutine_TitleAndDuration(t *testing.T) {
	var captured map[string]interface{}
	repo := &mocks.MockRoutineRepository{
		UpdateRoutineFunc: func(userID, routineID uint, updates map[string]interface{}) error {
			captured = updates
			return nil
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.UpdateRoutine(domain.UpdateRoutineRequest{
		ID: 1, UserID: 1, Title: ptrStr("New"), DurationMins: ptrInt(20),
	})
	require.NoError(t, err)
	assert.Equal(t, "New", captured["title"])
	assert.Equal(t, 20, captured["duration_mins"])
	_, hasGoalType := captured["goal_type"]
	assert.False(t, hasGoalType)
	_, hasGoalTarget := captured["goal_target"]
	assert.False(t, hasGoalTarget)
	_, hasRP := captured["reset_period"]
	assert.False(t, hasRP)
}

func TestService_UpdateRoutine_SetsGoalAndResetPeriod(t *testing.T) {
	var captured map[string]interface{}
	repo := &mocks.MockRoutineRepository{
		UpdateRoutineFunc: func(userID, routineID uint, updates map[string]interface{}) error {
			captured = updates
			return nil
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.UpdateRoutine(domain.UpdateRoutineRequest{
		ID: 1, UserID: 1,
		Goal:        &domain.Goal{Type: domain.GoalTypeTime, Target: 120},
		ResetPeriod: ptrStr(domain.ResetPeriodWeekly),
	})
	require.NoError(t, err)
	assert.Equal(t, "time", captured["goal_type"])
	assert.Equal(t, 120, captured["goal_target"])
	assert.Equal(t, "weekly", captured["reset_period"])
}

func TestService_UpdateRoutine_SwitchesGoalType(t *testing.T) {
	// Switching from a time goal to a count goal overwrites both columns
	// in a single update — no special-casing required.
	var captured map[string]interface{}
	repo := &mocks.MockRoutineRepository{
		UpdateRoutineFunc: func(userID, routineID uint, updates map[string]interface{}) error {
			captured = updates
			return nil
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.UpdateRoutine(domain.UpdateRoutineRequest{
		ID: 1, UserID: 1,
		Goal: &domain.Goal{Type: domain.GoalTypeCount, Target: 50},
	})
	require.NoError(t, err)
	assert.Equal(t, "count", captured["goal_type"])
	assert.Equal(t, 50, captured["goal_target"])
}

func TestService_UpdateRoutine_ZeroTargetClearsGoalAndResetPeriod(t *testing.T) {
	// FE sends Goal.Target=0 with no ResetPeriod -> demote back to a regular routine.
	// We expect goal_type, goal_target, and reset_period all cleared so a stranded
	// weekly/monthly value is wiped.
	var captured map[string]interface{}
	repo := &mocks.MockRoutineRepository{
		UpdateRoutineFunc: func(userID, routineID uint, updates map[string]interface{}) error {
			captured = updates
			return nil
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.UpdateRoutine(domain.UpdateRoutineRequest{
		ID: 1, UserID: 1, Goal: &domain.Goal{Target: 0},
	})
	require.NoError(t, err)

	require.Contains(t, captured, "goal_type")
	assert.Nil(t, captured["goal_type"])
	require.Contains(t, captured, "goal_target")
	assert.Nil(t, captured["goal_target"])
	require.Contains(t, captured, "reset_period")
	assert.Nil(t, captured["reset_period"])
}

func TestService_UpdateRoutine_ZeroTargetWithExplicitResetPeriod_RespectsCaller(t *testing.T) {
	// If the caller clears the goal AND sets a reset_period explicitly, honour
	// the caller's reset_period rather than overriding it to nil.
	var captured map[string]interface{}
	repo := &mocks.MockRoutineRepository{
		UpdateRoutineFunc: func(userID, routineID uint, updates map[string]interface{}) error {
			captured = updates
			return nil
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.UpdateRoutine(domain.UpdateRoutineRequest{
		ID: 1, UserID: 1,
		Goal:        &domain.Goal{Target: 0},
		ResetPeriod: ptrStr(domain.ResetPeriodWeekly),
	})
	require.NoError(t, err)
	assert.Nil(t, captured["goal_type"])
	assert.Nil(t, captured["goal_target"])
	assert.Equal(t, "weekly", captured["reset_period"])
}

func TestService_UpdateRoutine_EmptyResetPeriodNormalisedToNil(t *testing.T) {
	var captured map[string]interface{}
	repo := &mocks.MockRoutineRepository{
		UpdateRoutineFunc: func(userID, routineID uint, updates map[string]interface{}) error {
			captured = updates
			return nil
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.UpdateRoutine(domain.UpdateRoutineRequest{
		ID: 1, UserID: 1, ResetPeriod: ptrStr(""),
	})
	require.NoError(t, err)
	require.Contains(t, captured, "reset_period")
	assert.Nil(t, captured["reset_period"])
}

func TestService_UpdateRoutine_OneOffResetPeriodNormalisedToNil(t *testing.T) {
	var captured map[string]interface{}
	repo := &mocks.MockRoutineRepository{
		UpdateRoutineFunc: func(userID, routineID uint, updates map[string]interface{}) error {
			captured = updates
			return nil
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.UpdateRoutine(domain.UpdateRoutineRequest{
		ID: 1, UserID: 1, ResetPeriod: ptrStr(domain.ResetPeriodOneOff),
	})
	require.NoError(t, err)
	require.Contains(t, captured, "reset_period")
	assert.Nil(t, captured["reset_period"])
}

func TestService_UpdateRoutine_RepoError(t *testing.T) {
	repo := &mocks.MockRoutineRepository{
		UpdateRoutineFunc: func(userID, routineID uint, updates map[string]interface{}) error {
			return errors.New("db error")
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.UpdateRoutine(domain.UpdateRoutineRequest{
		ID: 1, UserID: 1, Title: ptrStr("New"),
	})
	assert.Error(t, err)
}

// --- GetRoutines ---

func TestService_GetRoutines_PassesThroughStats(t *testing.T) {
	repo := &mocks.MockRoutineRepository{
		GetRoutinesByUserIDFunc: func(userID uint) ([]domain.RoutineWithStats, error) {
			assert.Equal(t, uint(1), userID)
			return []domain.RoutineWithStats{
				{
					Routine:        domain.Routine{Title: "Read", DurationMins: 15},
					ScheduledMins:  30,
					CompletedMins:  15,
					InstanceCount:  2,
					CompletedCount: 1,
				},
			}, nil
		},
	}
	svc := newRoutineService(repo)

	resp, err := svc.GetRoutines(domain.GetRoutineRequest{UserID: 1})
	require.NoError(t, err)
	require.Len(t, resp.Routines, 1)
	assert.Equal(t, 30, resp.Routines[0].ScheduledMins)
	assert.Equal(t, 1, resp.Routines[0].CompletedCount)
}

func TestService_GetRoutines_RepoError(t *testing.T) {
	repo := &mocks.MockRoutineRepository{
		GetRoutinesByUserIDFunc: func(userID uint) ([]domain.RoutineWithStats, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.GetRoutines(domain.GetRoutineRequest{UserID: 1})
	assert.Error(t, err)
}

// --- CreateRoutineInstance ---

func TestService_CreateRoutineInstance_PassesDurationOverride(t *testing.T) {
	now := time.Now()
	var captured domain.RoutineInstance
	repo := &mocks.MockRoutineRepository{
		CreateInstanceFunc: func(ri domain.RoutineInstance) (domain.RoutineInstance, error) {
			captured = ri
			ri.ID = 9
			return ri, nil
		},
	}
	svc := newRoutineService(repo)

	resp, err := svc.CreateRoutineInstance(domain.CreateRoutineInstanceRequest{
		UserID:       1,
		RoutineID:    5,
		Date:         now,
		DurationMins: ptrInt(45),
	})
	require.NoError(t, err)

	assert.Equal(t, uint(9), resp.ID)
	assert.Equal(t, "needsAction", captured.Status)
	require.NotNil(t, captured.DurationMins)
	assert.Equal(t, 45, *captured.DurationMins)
	require.NotNil(t, resp.DurationMins)
	assert.Equal(t, 45, *resp.DurationMins)
}

func TestService_CreateRoutineInstance_NilDurationMeansInherit(t *testing.T) {
	var captured domain.RoutineInstance
	repo := &mocks.MockRoutineRepository{
		CreateInstanceFunc: func(ri domain.RoutineInstance) (domain.RoutineInstance, error) {
			captured = ri
			return ri, nil
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.CreateRoutineInstance(domain.CreateRoutineInstanceRequest{
		UserID: 1, RoutineID: 5, Date: time.Now(),
	})
	require.NoError(t, err)
	assert.Nil(t, captured.DurationMins)
}
