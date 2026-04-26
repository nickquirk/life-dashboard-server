# Plan: Fix phantom `scheduledMins` on freshly-created timed goals

## Background

When a user creates or converts a routine to a timed goal with `resetPeriod = 'one_off'` (the UI default), the progress bar immediately shows non-zero scheduled progress even though no instances exist. The cause is in the `GetRoutinesByUserID` SQL query, where the `scheduled_mins` aggregation lacks an "instance exists" guard that the sibling metrics happen to have.

Symptom only appears for `reset_period IS NULL` (one-off). Weekly/monthly resets are accidentally safe because their `inPeriod` predicate references `ri.date` and resolves to `NULL` when no instance exists, falling through to the ELSE branch.

## Root cause

`internal/repository/routines.go`, around line 2466:

```go
COALESCE(SUM(CASE WHEN `+inPeriod+`
    THEN COALESCE(ri.duration_mins, r.duration_mins)
    ELSE 0 END), 0) AS scheduled_mins,
```

`LEFT JOIN routine_instances` on a routine with no instances produces one row with all `ri.*` columns NULL. For one-off routines, `inPeriod` is `r.reset_period IS NULL` — true regardless of `ri.*`. The THEN clause then evaluates `COALESCE(NULL, r.duration_mins) = r.duration_mins`, and SUM over the phantom row yields `r.duration_mins`. So a brand-new one-off goal with `durationMins: 15` reports `scheduledMins = 15`.

The other three metrics survive by coincidence:

- `instance_count` explicitly checks `ri.id IS NOT NULL`.
- `completed_mins` and `completed_count` filter on `ri.status = 'completed'`, and `NULL = 'completed'` resolves to `NULL` (treated as falsy by CASE).

## Tasks

### 1. Fix the `scheduled_mins` aggregation

**File:** `internal/repository/routines.go`

In the `Select(...)` block in `GetRoutinesByUserID`, add the same `ri.id IS NOT NULL` guard `instance_count` already uses:

```go
COALESCE(SUM(CASE WHEN ri.id IS NOT NULL AND `+inPeriod+`
    THEN COALESCE(ri.duration_mins, r.duration_mins)
    ELSE 0 END), 0) AS scheduled_mins,
```

This is the only line strictly required to fix the reported bug.

### 2. Add the same guard to `completed_mins` and `completed_count`

Same file. The current expressions rely on `ri.status = 'completed'` to implicitly filter out phantom rows via NULL propagation. That works today, but it's a non-obvious property — the next refactor of this query is unlikely to preserve it. Make the intent explicit:

```go
COALESCE(SUM(CASE WHEN ri.id IS NOT NULL AND ri.status = 'completed' AND `+inPeriod+`
    THEN COALESCE(ri.duration_mins, r.duration_mins)
    ELSE 0 END), 0) AS completed_mins,
...
COALESCE(SUM(CASE WHEN ri.id IS NOT NULL AND ri.status = 'completed' AND `+inPeriod+`
    THEN 1 ELSE 0 END), 0) AS completed_count
```

This is a no-op on observable behavior — verify with the test suite from task 3.

### 3. Add repository test coverage for `GetRoutinesByUserID`

There are currently no repository-level tests in `internal/repository/`. Service-level tests use mocks, which is why the SQL bug went undetected. Introduce a minimal pattern here.

**Recommended approach:** SQLite in-memory via `gorm.io/driver/sqlite`. Production is MySQL, but the constructs in this query (`CASE WHEN`, `COALESCE`, `LEFT JOIN`, `SUM`, `GROUP BY`, `IS NULL`, `IS NOT NULL`, integer comparisons against parameterised dates) behave identically across both dialects, and SQLite gives sub-second test runs with no infrastructure. If the team prefers fidelity to production, swap in dockertest/testcontainers with MySQL — but don't block on that for this fix.

**File (new):** `internal/repository/routines_test.go`

Setup:

```go
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
```

Add `gorm.io/driver/sqlite` to `go.mod` as a test-only dep (it is, in practice — only imported from `_test.go` files).

Required cases. Each one creates a fresh DB, inserts data, calls `GetRoutinesByUserID(userID)`, asserts on the returned `RoutineWithStats`:

1. **One-off routine with no instances** — the regression case.
   - Insert routine: `DurationMins=15`, `TargetTotalMins=ptr(60)`, `ResetPeriod=nil`.
   - Assert `ScheduledMins == 0`, `CompletedMins == 0`, `InstanceCount == 0`, `CompletedCount == 0`.
   - **This test fails on `main` and passes after task 1.**

2. **Weekly routine with no instances** — confirms the accidentally-safe path stays safe.
   - Same as above with `ResetPeriod=ptr("weekly")`.
   - All four metrics should be 0.

3. **One-off routine with one needsAction instance** — basic positive case.
   - Routine with `DurationMins=15`. One instance with status `needsAction`, `DurationMins=nil`.
   - Assert `ScheduledMins == 15`, `CompletedMins == 0`, `InstanceCount == 1`, `CompletedCount == 0`.

4. **One-off routine with one completed instance**.
   - Same setup, instance status `completed`.
   - Assert `ScheduledMins == 15`, `CompletedMins == 15`, `InstanceCount == 1`, `CompletedCount == 1`.

5. **Instance-level `DurationMins` override** — verifies the `COALESCE(ri.duration_mins, r.duration_mins)` precedence.
   - Routine with `DurationMins=15`. One instance with `DurationMins=ptr(45)`, status `needsAction`.
   - Assert `ScheduledMins == 45`.

6. **Weekly routine, instance from previous week** — verifies period filtering.
   - Routine `ResetPeriod=ptr("weekly")`, `DurationMins=20`. One instance dated 8 days before `time.Now()`, status `needsAction`.
   - Assert `ScheduledMins == 0`, `InstanceCount == 0`. (Instance exists in DB but is outside the period.)

7. **Weekly routine, instance from this week** — same setup, instance dated yesterday or today. Expect `ScheduledMins == 20`, `InstanceCount == 1`.

8. **Soft-deleted instance is excluded.** Insert an instance, then `db.Delete(&instance)` (soft delete via `gorm.DeletedAt`). Re-query. Expect counts to ignore it.

Tests 1, 2, and 8 are the ones most likely to catch future regressions; the others document positive behaviour.

### 4. Make `CreateRoutine` response self-consistent

**File:** `internal/service/routines.go`, around line 4894.

Currently:

```go
return domain.CreateRoutineResponse{
    ID:              created.ID,
    Title:           created.Title,
    DurationMins:    created.DurationMins,
    TargetTotalMins: req.TargetTotalMins,   // ← reads from request
    ResetPeriod:     created.ResetPeriod,
}, nil
```

Switch the `TargetTotalMins` line to read from `created.TargetTotalMins`. Behaviour-equivalent today (GORM doesn't sanitise), but the inconsistency is a small landmine if the repo ever gains insert-time normalisation.

### 5. Decide on `instanceCount` / `completedCount`

These are computed by the SQL, transported through the API, declared in `src/lib/types.ts`, populated in test mocks — and never rendered. Two options, pick one:

**Option A — wire them up.** In `RoutineCard.tsx`, add a small "n of m done" line beneath the progress bar for finite goals. Trivial change; the data is already there.

**Option B — drop them.** Remove `InstanceCount` / `CompletedCount` from `domain.RoutineWithStats` and the `Select(...)` clause; remove `instanceCount` / `completedCount` from the `Routine` type and mocks.

A is the lower-risk choice (no breaking changes, gives users a useful counter). B is fine if there's a deliberate decision not to surface them. Don't leave it as-is — dead end-to-end fields are how the symmetry that hid this bug got introduced in the first place.

## Verification checklist

- [ ] `go test ./internal/repository/...` — new tests pass after task 1.
- [ ] On task 1 alone (without 2): all four new metric assertions on the one-off-empty case pass.
- [ ] On task 2 alone (without 1): no test changes behaviour — confirms it's a no-op refactor.
- [ ] Manually: create a new timed goal with `targetTotalMins=60`, `durationMins=15`, default reset period. Progress bar shows 0/60, not 15/60.
- [ ] Drag one instance onto the calendar. Progress bar shows 15/60.
- [ ] Mark the instance complete. The completed (darker) bar shows 15/60.
- [ ] Repeat with `resetPeriod=weekly`. Confirm progress matches expectations.
- [ ] If task 5A picked: counter renders correctly for 0, 1, multiple instances.

## Out of scope

- The frontend optimistic-update path in `useRoutineInstances.ts` doesn't bump `routine.scheduledMins` until the server resync lands. With the SQL fix, the resync becomes correct, so the bar transitions `0 → N` rather than `15 → N`. The lag itself isn't part of this fix.
- The SQL placeholder count (8 `?`s, 8 args) is correct — leave it alone.
- The ghost-routine timezone bug from the previous plan — separate concern, separate PR.