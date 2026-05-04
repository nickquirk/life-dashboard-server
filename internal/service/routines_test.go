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
		UserID:          1,
		Title:           "Read",
		DurationMins:    15,
		TargetTotalMins: ptrInt(60),
		ResetPeriod:     ptrStr(domain.ResetPeriodWeekly),
	}
	resp, err := svc.CreateRoutine(req)
	require.NoError(t, err)

	assert.Equal(t, uint(7), resp.ID)
	assert.Equal(t, "Read", resp.Title)
	assert.Equal(t, 15, resp.DurationMins)
	require.NotNil(t, resp.TargetTotalMins)
	assert.Equal(t, 60, *resp.TargetTotalMins)
	require.NotNil(t, resp.ResetPeriod)
	assert.Equal(t, domain.ResetPeriodWeekly, *resp.ResetPeriod)

	assert.Equal(t, uint(1), captured.UserID)
	assert.Equal(t, "Read", captured.Title)
	require.NotNil(t, captured.TargetTotalMins)
	assert.Equal(t, 60, *captured.TargetTotalMins)
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
	_, hasTTM := captured["target_total_mins"]
	assert.False(t, hasTTM)
	_, hasRP := captured["reset_period"]
	assert.False(t, hasRP)
}

func TestService_UpdateRoutine_SetsTargetAndResetPeriod(t *testing.T) {
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
		TargetTotalMins: ptrInt(120),
		ResetPeriod:     ptrStr(domain.ResetPeriodWeekly),
	})
	require.NoError(t, err)
	assert.Equal(t, 120, captured["target_total_mins"])
	assert.Equal(t, "weekly", captured["reset_period"])
}

func TestService_UpdateRoutine_ZeroTargetClearsTargetAndResetPeriod(t *testing.T) {
	// FE sends TTM=0 with no ResetPeriod -> demote back to a regular routine.
	// We expect both fields cleared so a stranded weekly/monthly value is wiped.
	var captured map[string]interface{}
	repo := &mocks.MockRoutineRepository{
		UpdateRoutineFunc: func(userID, routineID uint, updates map[string]interface{}) error {
			captured = updates
			return nil
		},
	}
	svc := newRoutineService(repo)

	_, err := svc.UpdateRoutine(domain.UpdateRoutineRequest{
		ID: 1, UserID: 1, TargetTotalMins: ptrInt(0),
	})
	require.NoError(t, err)

	require.Contains(t, captured, "target_total_mins")
	assert.Nil(t, captured["target_total_mins"])
	require.Contains(t, captured, "reset_period")
	assert.Nil(t, captured["reset_period"])
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
