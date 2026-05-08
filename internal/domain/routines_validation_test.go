package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptrInt(v int) *int       { return &v }
func ptrStr(v string) *string { return &v }

// --- CreateRoutineRequest.Validate ---

func TestCreateRoutineRequest_Valid_Minimal(t *testing.T) {
	req := CreateRoutineRequest{Title: "Read", DurationMins: 15}
	require.NoError(t, req.Validate())
}

func TestCreateRoutineRequest_Valid_AllFields(t *testing.T) {
	req := CreateRoutineRequest{
		Title:        "Run",
		DurationMins: 30,
		Goal:         &Goal{Type: GoalTypeTime, Target: 120},
		ResetPeriod:  ptrStr(ResetPeriodWeekly),
	}
	require.NoError(t, req.Validate())
}

func TestCreateRoutineRequest_Valid_CountGoal(t *testing.T) {
	req := CreateRoutineRequest{
		Title:        "Pushups",
		DurationMins: 10,
		Goal:         &Goal{Type: GoalTypeCount, Target: 50},
	}
	require.NoError(t, req.Validate())
}

func TestCreateRoutineRequest_TrimsTitle(t *testing.T) {
	req := CreateRoutineRequest{Title: "  Read  ", DurationMins: 15}
	require.NoError(t, req.Validate())
	assert.Equal(t, "Read", req.Title)
}

func TestCreateRoutineRequest_EmptyTitle(t *testing.T) {
	req := CreateRoutineRequest{Title: "   ", DurationMins: 15}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
	assert.Contains(t, err.Error(), "title is required")
}

func TestCreateRoutineRequest_TitleTooLong(t *testing.T) {
	req := CreateRoutineRequest{Title: strings.Repeat("a", 201), DurationMins: 15}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

func TestCreateRoutineRequest_DurationOutOfRange(t *testing.T) {
	cases := []int{0, -5, 24*60 + 1}
	for _, d := range cases {
		req := CreateRoutineRequest{Title: "Read", DurationMins: d}
		err := req.Validate()
		require.Error(t, err, "expected error for duration=%d", d)
		assert.True(t, errors.Is(err, ErrInvalidInput))
	}
}

func TestCreateRoutineRequest_TimeGoalOutOfRange(t *testing.T) {
	cases := []int{-1, 30*24*60 + 1}
	for _, ttm := range cases {
		req := CreateRoutineRequest{
			Title:        "Read",
			DurationMins: 15,
			Goal:         &Goal{Type: GoalTypeTime, Target: ttm},
		}
		err := req.Validate()
		require.Error(t, err, "expected error for target=%d", ttm)
		assert.True(t, errors.Is(err, ErrInvalidInput))
	}
}

func TestCreateRoutineRequest_CountGoalOutOfRange(t *testing.T) {
	cases := []int{-1, 1001}
	for _, ttm := range cases {
		req := CreateRoutineRequest{
			Title:        "Pushups",
			DurationMins: 10,
			Goal:         &Goal{Type: GoalTypeCount, Target: ttm},
		}
		err := req.Validate()
		require.Error(t, err, "expected error for target=%d", ttm)
		assert.True(t, errors.Is(err, ErrInvalidInput))
	}
}

func TestCreateRoutineRequest_InvalidGoalType(t *testing.T) {
	req := CreateRoutineRequest{
		Title:        "Read",
		DurationMins: 15,
		Goal:         &Goal{Type: "distance", Target: 5},
	}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

func TestCreateRoutineRequest_GoalTargetZeroAccepted(t *testing.T) {
	// Target == 0 is the "clear goal" sentinel; Type is ignored.
	req := CreateRoutineRequest{
		Title:        "Read",
		DurationMins: 15,
		Goal:         &Goal{Target: 0},
	}
	require.NoError(t, req.Validate())
}

func TestCreateRoutineRequest_InvalidResetPeriod(t *testing.T) {
	req := CreateRoutineRequest{
		Title:        "Read",
		DurationMins: 15,
		ResetPeriod:  ptrStr("daily"),
	}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

func TestCreateRoutineRequest_AcceptsAllValidResetPeriods(t *testing.T) {
	for _, rp := range []string{"", ResetPeriodOneOff, ResetPeriodWeekly, ResetPeriodMonthly} {
		req := CreateRoutineRequest{Title: "Read", DurationMins: 15, ResetPeriod: ptrStr(rp)}
		assert.NoError(t, req.Validate(), "resetPeriod=%q", rp)
	}
}

// --- UpdateRoutineRequest.Validate ---

func TestUpdateRoutineRequest_NoFields_OK(t *testing.T) {
	req := UpdateRoutineRequest{}
	require.NoError(t, req.Validate())
}

func TestUpdateRoutineRequest_TrimsTitle(t *testing.T) {
	title := "  Updated  "
	req := UpdateRoutineRequest{Title: &title}
	require.NoError(t, req.Validate())
	assert.Equal(t, "Updated", *req.Title)
}

func TestUpdateRoutineRequest_EmptyTitleRejected(t *testing.T) {
	title := "   "
	req := UpdateRoutineRequest{Title: &title}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

func TestUpdateRoutineRequest_TitleTooLong(t *testing.T) {
	title := strings.Repeat("a", 201)
	req := UpdateRoutineRequest{Title: &title}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

func TestUpdateRoutineRequest_DurationOutOfRange(t *testing.T) {
	for _, d := range []int{0, -1, 24*60 + 1} {
		req := UpdateRoutineRequest{DurationMins: ptrInt(d)}
		err := req.Validate()
		require.Error(t, err, "duration=%d", d)
		assert.True(t, errors.Is(err, ErrInvalidInput))
	}
}

func TestUpdateRoutineRequest_NilGoalAccepted(t *testing.T) {
	// nil Goal means "no change".
	req := UpdateRoutineRequest{Goal: nil}
	require.NoError(t, req.Validate())
}

func TestUpdateRoutineRequest_GoalTargetZeroAccepted(t *testing.T) {
	// Target == 0 is the "clear goal" sentinel; Type is ignored.
	req := UpdateRoutineRequest{Goal: &Goal{Target: 0}}
	require.NoError(t, req.Validate())
}

func TestUpdateRoutineRequest_TimeGoalOutOfRange(t *testing.T) {
	for _, ttm := range []int{-1, 30*24*60 + 1} {
		req := UpdateRoutineRequest{Goal: &Goal{Type: GoalTypeTime, Target: ttm}}
		err := req.Validate()
		require.Error(t, err, "target=%d", ttm)
		assert.True(t, errors.Is(err, ErrInvalidInput))
	}
}

func TestUpdateRoutineRequest_CountGoalOutOfRange(t *testing.T) {
	for _, ttm := range []int{-1, 1001} {
		req := UpdateRoutineRequest{Goal: &Goal{Type: GoalTypeCount, Target: ttm}}
		err := req.Validate()
		require.Error(t, err, "target=%d", ttm)
		assert.True(t, errors.Is(err, ErrInvalidInput))
	}
}

func TestUpdateRoutineRequest_InvalidGoalType(t *testing.T) {
	req := UpdateRoutineRequest{Goal: &Goal{Type: "distance", Target: 5}}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

func TestUpdateRoutineRequest_InvalidResetPeriod(t *testing.T) {
	req := UpdateRoutineRequest{ResetPeriod: ptrStr("yearly")}
	err := req.Validate()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput))
}

func TestUpdateRoutineRequest_EmptyResetPeriodAccepted(t *testing.T) {
	req := UpdateRoutineRequest{ResetPeriod: ptrStr("")}
	require.NoError(t, req.Validate())
}
