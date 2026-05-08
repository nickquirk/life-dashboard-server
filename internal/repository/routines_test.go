package repository

import (
	"testing"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const testUserID uint = 1

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.Routine{}, &domain.RoutineInstance{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newRepo(t *testing.T) (*GormRoutineRepository, *gorm.DB) {
	t.Helper()
	db := newTestDB(t)
	return &GormRoutineRepository{Db: db}, db
}

func ptrInt(v int) *int       { return &v }
func ptrStr(v string) *string { return &v }

func seedRoutine(t *testing.T, db *gorm.DB, r domain.Routine) domain.Routine {
	t.Helper()
	if r.UserID == 0 {
		r.UserID = testUserID
	}
	require.NoError(t, db.Create(&r).Error)
	return r
}

func seedInstance(t *testing.T, db *gorm.DB, ri domain.RoutineInstance) domain.RoutineInstance {
	t.Helper()
	if ri.UserID == 0 {
		ri.UserID = testUserID
	}
	if ri.Status == "" {
		ri.Status = "needsAction"
	}
	require.NoError(t, db.Create(&ri).Error)
	return ri
}

func getOnly(t *testing.T, repo *GormRoutineRepository) domain.RoutineWithStats {
	t.Helper()
	out, err := repo.GetRoutinesByUserID(testUserID)
	require.NoError(t, err)
	require.Len(t, out, 1)
	return out[0]
}

// 1. One-off routine with no instances — the regression case.
func TestGetRoutinesByUserID_OneOff_NoInstances(t *testing.T) {
	repo, db := newRepo(t)
	seedRoutine(t, db, domain.Routine{
		Title:        "Read",
		DurationMins: 15,
		GoalType:     ptrStr(string(domain.GoalTypeTime)),
		GoalTarget:   ptrInt(60),
		ResetPeriod:  nil,
	})

	got := getOnly(t, repo)
	assert.Equal(t, 0, got.ScheduledMins)
	assert.Equal(t, 0, got.CompletedMins)
	assert.Equal(t, 0, got.InstanceCount)
	assert.Equal(t, 0, got.CompletedCount)
	require.NotNil(t, got.GoalType)
	assert.Equal(t, "time", *got.GoalType)
	require.NotNil(t, got.GoalTarget)
	assert.Equal(t, 60, *got.GoalTarget)
}

// 2. Weekly routine with no instances — accidentally-safe path stays safe.
func TestGetRoutinesByUserID_Weekly_NoInstances(t *testing.T) {
	repo, db := newRepo(t)
	seedRoutine(t, db, domain.Routine{
		Title:        "Run",
		DurationMins: 15,
		GoalType:     ptrStr(string(domain.GoalTypeTime)),
		GoalTarget:   ptrInt(60),
		ResetPeriod:  ptrStr("weekly"),
	})

	got := getOnly(t, repo)
	assert.Equal(t, 0, got.ScheduledMins)
	assert.Equal(t, 0, got.CompletedMins)
	assert.Equal(t, 0, got.InstanceCount)
	assert.Equal(t, 0, got.CompletedCount)
	require.NotNil(t, got.GoalType)
	assert.Equal(t, "time", *got.GoalType)
	require.NotNil(t, got.GoalTarget)
	assert.Equal(t, 60, *got.GoalTarget)
}

// 2b. Count goal storage columns round-trip through GetRoutinesByUserID.
func TestGetRoutinesByUserID_CountGoalColumnsReturned(t *testing.T) {
	repo, db := newRepo(t)
	seedRoutine(t, db, domain.Routine{
		Title:        "Pushups",
		DurationMins: 5,
		GoalType:     ptrStr(string(domain.GoalTypeCount)),
		GoalTarget:   ptrInt(100),
	})

	got := getOnly(t, repo)
	require.NotNil(t, got.GoalType)
	assert.Equal(t, "count", *got.GoalType)
	require.NotNil(t, got.GoalTarget)
	assert.Equal(t, 100, *got.GoalTarget)
}

// 2c. A routine with no goal columns leaves GoalType / GoalTarget nil.
func TestGetRoutinesByUserID_NoGoal_ColumnsAreNil(t *testing.T) {
	repo, db := newRepo(t)
	seedRoutine(t, db, domain.Routine{
		Title:        "Read",
		DurationMins: 15,
	})

	got := getOnly(t, repo)
	assert.Nil(t, got.GoalType)
	assert.Nil(t, got.GoalTarget)
}

// 3. One-off routine with one needsAction instance.
func TestGetRoutinesByUserID_OneOff_OneNeedsAction(t *testing.T) {
	repo, db := newRepo(t)
	r := seedRoutine(t, db, domain.Routine{
		Title:        "Read",
		DurationMins: 15,
	})
	seedInstance(t, db, domain.RoutineInstance{
		RoutineID: r.ID,
		Date:      time.Now(),
		Status:    "needsAction",
	})

	got := getOnly(t, repo)
	assert.Equal(t, 15, got.ScheduledMins)
	assert.Equal(t, 0, got.CompletedMins)
	assert.Equal(t, 1, got.InstanceCount)
	assert.Equal(t, 0, got.CompletedCount)
}

// 4. One-off routine with one completed instance.
func TestGetRoutinesByUserID_OneOff_OneCompleted(t *testing.T) {
	repo, db := newRepo(t)
	r := seedRoutine(t, db, domain.Routine{
		Title:        "Read",
		DurationMins: 15,
	})
	seedInstance(t, db, domain.RoutineInstance{
		RoutineID: r.ID,
		Date:      time.Now(),
		Status:    "completed",
	})

	got := getOnly(t, repo)
	assert.Equal(t, 15, got.ScheduledMins)
	assert.Equal(t, 15, got.CompletedMins)
	assert.Equal(t, 1, got.InstanceCount)
	assert.Equal(t, 1, got.CompletedCount)
}

// 5. Instance-level DurationMins override takes precedence.
func TestGetRoutinesByUserID_InstanceDurationOverride(t *testing.T) {
	repo, db := newRepo(t)
	r := seedRoutine(t, db, domain.Routine{
		Title:        "Read",
		DurationMins: 15,
	})
	seedInstance(t, db, domain.RoutineInstance{
		RoutineID:    r.ID,
		Date:         time.Now(),
		Status:       "needsAction",
		DurationMins: ptrInt(45),
	})

	got := getOnly(t, repo)
	assert.Equal(t, 45, got.ScheduledMins)
}

// 6. Weekly routine, instance from previous week is excluded by period filter.
func TestGetRoutinesByUserID_Weekly_PreviousWeekInstanceExcluded(t *testing.T) {
	repo, db := newRepo(t)
	r := seedRoutine(t, db, domain.Routine{
		Title:        "Run",
		DurationMins: 20,
		ResetPeriod:  ptrStr("weekly"),
	})
	seedInstance(t, db, domain.RoutineInstance{
		RoutineID: r.ID,
		Date:      time.Now().AddDate(0, 0, -8),
		Status:    "needsAction",
	})

	got := getOnly(t, repo)
	assert.Equal(t, 0, got.ScheduledMins)
	assert.Equal(t, 0, got.InstanceCount)
}

// 7. Weekly routine, instance from this week is included.
func TestGetRoutinesByUserID_Weekly_ThisWeekInstanceIncluded(t *testing.T) {
	repo, db := newRepo(t)
	r := seedRoutine(t, db, domain.Routine{
		Title:        "Run",
		DurationMins: 20,
		ResetPeriod:  ptrStr("weekly"),
	})
	seedInstance(t, db, domain.RoutineInstance{
		RoutineID: r.ID,
		Date:      time.Now(),
		Status:    "needsAction",
	})

	got := getOnly(t, repo)
	assert.Equal(t, 20, got.ScheduledMins)
	assert.Equal(t, 1, got.InstanceCount)
}

// 8. Routines belonging to another user are not returned.
func TestGetRoutinesByUserID_MultiUserIsolation(t *testing.T) {
	const otherUserID uint = 2

	repo, db := newRepo(t)
	seedRoutine(t, db, domain.Routine{UserID: testUserID, Title: "User1 Routine", DurationMins: 30})
	seedRoutine(t, db, domain.Routine{UserID: otherUserID, Title: "User2 Routine", DurationMins: 30})

	out, err := repo.GetRoutinesByUserID(testUserID)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "User1 Routine", out[0].Title)
}

// 9a. Archived routine is excluded from GetRoutinesByUserID.
func TestGetRoutinesByUserID_ArchivedExcluded(t *testing.T) {
	repo, db := newRepo(t)
	seedRoutine(t, db, domain.Routine{
		Title:        "Active",
		DurationMins: 15,
	})
	seedRoutine(t, db, domain.Routine{
		Title:        "Archived",
		DurationMins: 15,
		IsArchived:   true,
	})

	out, err := repo.GetRoutinesByUserID(testUserID)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "Active", out[0].Title)
}

// 9b. Archived routine's instances are still returned by GetInstancesByUserID.
// Design point: archiving hides the template from the sidebar without
// destroying historical instances.
func TestGetInstancesByUserID_ArchivedRoutineInstancesRetained(t *testing.T) {
	repo, db := newRepo(t)
	r := seedRoutine(t, db, domain.Routine{
		Title:        "Archived",
		DurationMins: 30,
		IsArchived:   true,
	})
	seedInstance(t, db, domain.RoutineInstance{
		RoutineID: r.ID,
		Date:      time.Now(),
		Status:    "completed",
	})

	now := time.Now()
	out, err := repo.GetInstancesByUserID(
		testUserID,
		now.AddDate(0, 0, -1),
		now.AddDate(0, 0, 1),
	)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, r.ID, out[0].RoutineID)
	assert.Equal(t, "completed", out[0].Status)
	assert.Equal(t, "Archived", out[0].Routine.Title)
}

// 9c. Unarchiving restores the routine to GetRoutinesByUserID with its
// original stats intact (instances were never touched).
func TestGetRoutinesByUserID_UnarchiveRestoresStats(t *testing.T) {
	repo, db := newRepo(t)
	r := seedRoutine(t, db, domain.Routine{
		Title:        "Read",
		DurationMins: 15,
	})
	seedInstance(t, db, domain.RoutineInstance{
		RoutineID: r.ID,
		Date:      time.Now(),
		Status:    "completed",
	})
	seedInstance(t, db, domain.RoutineInstance{
		RoutineID: r.ID,
		Date:      time.Now(),
		Status:    "needsAction",
	})

	require.NoError(t, repo.UpdateRoutine(testUserID, r.ID, map[string]interface{}{
		"is_archived": true,
	}))
	hidden, err := repo.GetRoutinesByUserID(testUserID)
	require.NoError(t, err)
	require.Len(t, hidden, 0)

	require.NoError(t, repo.UpdateRoutine(testUserID, r.ID, map[string]interface{}{
		"is_archived": false,
	}))
	got := getOnly(t, repo)
	assert.Equal(t, "Read", got.Title)
	assert.Equal(t, 30, got.ScheduledMins)
	assert.Equal(t, 15, got.CompletedMins)
	assert.Equal(t, 2, got.InstanceCount)
	assert.Equal(t, 1, got.CompletedCount)
}

// 9d. UpdateRoutine sets goal_type + goal_target columns on an existing routine.
func TestUpdateRoutine_SetsGoalColumns(t *testing.T) {
	repo, db := newRepo(t)
	r := seedRoutine(t, db, domain.Routine{
		Title:        "Read",
		DurationMins: 15,
	})

	require.NoError(t, repo.UpdateRoutine(testUserID, r.ID, map[string]interface{}{
		"goal_type":   "time",
		"goal_target": 60,
	}))

	got := getOnly(t, repo)
	require.NotNil(t, got.GoalType)
	assert.Equal(t, "time", *got.GoalType)
	require.NotNil(t, got.GoalTarget)
	assert.Equal(t, 60, *got.GoalTarget)
}

// 9e. UpdateRoutine clears goal_type + goal_target back to NULL.
func TestUpdateRoutine_ClearsGoalColumns(t *testing.T) {
	repo, db := newRepo(t)
	r := seedRoutine(t, db, domain.Routine{
		Title:        "Read",
		DurationMins: 15,
		GoalType:     ptrStr(string(domain.GoalTypeTime)),
		GoalTarget:   ptrInt(60),
		ResetPeriod:  ptrStr("weekly"),
	})

	require.NoError(t, repo.UpdateRoutine(testUserID, r.ID, map[string]interface{}{
		"goal_type":    nil,
		"goal_target":  nil,
		"reset_period": nil,
	}))

	got := getOnly(t, repo)
	assert.Nil(t, got.GoalType)
	assert.Nil(t, got.GoalTarget)
	assert.Nil(t, got.ResetPeriod)
}

// 9f. UpdateRoutine can switch goal_type from time to count in one update.
func TestUpdateRoutine_SwitchesGoalType(t *testing.T) {
	repo, db := newRepo(t)
	r := seedRoutine(t, db, domain.Routine{
		Title:        "Pushups",
		DurationMins: 5,
		GoalType:     ptrStr(string(domain.GoalTypeTime)),
		GoalTarget:   ptrInt(30),
	})

	require.NoError(t, repo.UpdateRoutine(testUserID, r.ID, map[string]interface{}{
		"goal_type":   "count",
		"goal_target": 100,
	}))

	got := getOnly(t, repo)
	require.NotNil(t, got.GoalType)
	assert.Equal(t, "count", *got.GoalType)
	require.NotNil(t, got.GoalTarget)
	assert.Equal(t, 100, *got.GoalTarget)
}

// 10. Soft-deleted instance is excluded from all aggregates.
func TestGetRoutinesByUserID_SoftDeletedInstanceExcluded(t *testing.T) {
	repo, db := newRepo(t)
	r := seedRoutine(t, db, domain.Routine{
		Title:        "Read",
		DurationMins: 15,
	})
	ri := seedInstance(t, db, domain.RoutineInstance{
		RoutineID: r.ID,
		Date:      time.Now(),
		Status:    "completed",
	})
	require.NoError(t, db.Delete(&ri).Error)

	got := getOnly(t, repo)
	assert.Equal(t, 0, got.ScheduledMins)
	assert.Equal(t, 0, got.CompletedMins)
	assert.Equal(t, 0, got.InstanceCount)
	assert.Equal(t, 0, got.CompletedCount)
}
