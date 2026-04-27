package domain

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidInput is returned by request Validate() methods. Wrap with %w
// so callers can use errors.Is to check, and err.Error() carries the detail.
var ErrInvalidInput = errors.New("invalid input")

const (
	ResetPeriodOneOff  = "one_off"
	ResetPeriodWeekly  = "weekly"
	ResetPeriodMonthly = "monthly"
)

// "" is accepted as a synonym for "one_off" to preserve existing FE behaviour
// (a stale client may send empty strings to mean "clear"). Both map to nil
// in the repo.
var validResetPeriods = map[string]struct{}{
	"":                 {},
	ResetPeriodOneOff:  {},
	ResetPeriodWeekly:  {},
	ResetPeriodMonthly: {},
}

const (
	maxRoutineTitleLen     = 200
	maxRoutineDurationMins = 24 * 60      // 1 day — a single routine longer than that is almost certainly a typo
	maxRoutineTargetMins   = 30 * 24 * 60 // 30 days; generous headroom for monthly goals
)

func (r *CreateRoutineRequest) Validate() error {
	title := strings.TrimSpace(r.Title)
	if title == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if len(title) > maxRoutineTitleLen {
		return fmt.Errorf("%w: title must be %d characters or fewer", ErrInvalidInput, maxRoutineTitleLen)
	}
	r.Title = title // normalise so the repo stores the trimmed value

	if r.DurationMins <= 0 || r.DurationMins > maxRoutineDurationMins {
		return fmt.Errorf("%w: durationMins must be between 1 and %d", ErrInvalidInput, maxRoutineDurationMins)
	}
	if r.TargetTotalMins != nil && (*r.TargetTotalMins < 0 || *r.TargetTotalMins > maxRoutineTargetMins) {
		return fmt.Errorf("%w: targetTotalMins must be between 0 and %d", ErrInvalidInput, maxRoutineTargetMins)
	}
	if r.ResetPeriod != nil {
		if _, ok := validResetPeriods[*r.ResetPeriod]; !ok {
			return fmt.Errorf("%w: resetPeriod must be one of weekly, monthly, one_off", ErrInvalidInput)
		}
	}
	return nil
}

func (r *UpdateRoutineRequest) Validate() error {
	if r.Title != nil {
		title := strings.TrimSpace(*r.Title)
		if title == "" {
			return fmt.Errorf("%w: title cannot be empty", ErrInvalidInput)
		}
		if len(title) > maxRoutineTitleLen {
			return fmt.Errorf("%w: title must be %d characters or fewer", ErrInvalidInput, maxRoutineTitleLen)
		}
		*r.Title = title
	}
	if r.DurationMins != nil && (*r.DurationMins <= 0 || *r.DurationMins > maxRoutineDurationMins) {
		return fmt.Errorf("%w: durationMins must be between 1 and %d", ErrInvalidInput, maxRoutineDurationMins)
	}
	if r.TargetTotalMins != nil && (*r.TargetTotalMins < 0 || *r.TargetTotalMins > maxRoutineTargetMins) {
		return fmt.Errorf("%w: targetTotalMins must be between 0 and %d", ErrInvalidInput, maxRoutineTargetMins)
	}
	if r.ResetPeriod != nil {
		if _, ok := validResetPeriods[*r.ResetPeriod]; !ok {
			return fmt.Errorf("%w: resetPeriod must be one of weekly, monthly, one_off", ErrInvalidInput)
		}
	}
	return nil
}
