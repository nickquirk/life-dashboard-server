package repository

import (
	"encoding/json"
	"testing"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Bug repro: GetRoutinesByUserID returns a Go nil slice (not []) ---------
//
// Root cause. GetRoutinesByUserID declares `var routines []domain.RoutineWithStats`
// and returns it directly. When the query matches zero rows (the user has only
// archived / soft-deleted routines, or none at all) the slice is never appended
// to, so it stays nil. A nil []T marshals to JSON `null`, not `[]`.
//
// Downstream the service wraps it in GetRoutineResponse{Routines: nil} (no
// `omitempty`), the handler 200s it, and the frontend's `routines.length`
// throws "Cannot read properties of null (reading 'length')" because its
// `= []` destructuring default only catches `undefined`, never `null`.
//
// These tests fail today (the slice is nil) and pass once the repository
// initialises it with make([]domain.RoutineWithStats, 0) (or the service
// coalesces nil -> empty).

// 1. User with no routines at all: the empty-DB case.
func TestGetRoutinesByUserID_NoRoutines_ReturnsNonNilSlice(t *testing.T) {
	repo, _ := newRepo(t)

	out, err := repo.GetRoutinesByUserID(testUserID)
	require.NoError(t, err)
	require.Len(t, out, 0)

	// The crux: an empty result must be a non-nil empty slice so it
	// serialises to `[]`, never `null`.
	assert.NotNil(t, out, "empty result must be a non-nil slice, got nil (will marshal to JSON null)")

	b, err := json.Marshal(out)
	require.NoError(t, err)
	assert.Equal(t, "[]", string(b), "nil slice marshals to null; frontend routines.length then crashes")
}

//  2. The actual reported scenario: the user's only routine is archived, so the
//     `is_archived = false` filter removes it and the result set is empty.
//     This mirrors the DB snapshot row (id 15, "gfddfsfdgdfszeg",
//     is_archived = 1) that left this user with zero visible routines.
func TestGetRoutinesByUserID_OnlyArchivedRoutine_ReturnsNonNilSlice(t *testing.T) {
	repo, db := newRepo(t)

	seedRoutine(t, db, domain.Routine{
		Title:        "gfddfsfdgdfszeg",
		DurationMins: 15,
		GoalType:     ptrStr(string(domain.GoalTypeTime)),
		GoalTarget:   ptrInt(240),
		IsArchived:   true,
	})

	out, err := repo.GetRoutinesByUserID(testUserID)
	require.NoError(t, err)
	require.Len(t, out, 0, "archived routine is correctly filtered out")

	assert.NotNil(t, out, "all-archived user must still get [] not nil")

	b, err := json.Marshal(out)
	require.NoError(t, err)
	assert.Equal(t, "[]", string(b))
}

//  3. Same again via soft-delete, to show it is not specific to the archived
//     flag — any filter that empties the result set reproduces it.
func TestGetRoutinesByUserID_OnlySoftDeletedRoutine_ReturnsNonNilSlice(t *testing.T) {
	repo, db := newRepo(t)

	r := seedRoutine(t, db, domain.Routine{
		Title:        "Deleted",
		DurationMins: 15,
	})
	require.NoError(t, db.Delete(&r).Error) // gorm soft delete

	out, err := repo.GetRoutinesByUserID(testUserID)
	require.NoError(t, err)
	require.Len(t, out, 0)

	assert.NotNil(t, out, "soft-deleted-only user must still get [] not nil")
}
