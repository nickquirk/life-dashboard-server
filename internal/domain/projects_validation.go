package domain

import (
	"fmt"
	"strings"
)

const (
	maxProjectTitleLen = 200
	maxActionTitleLen  = 200
	maxActionNotesLen  = 10_000
	minDurationMins    = 1
	maxDurationMins    = 1440
	minBreakDownTitles = 2
)

func validateTitle(title *string, maxLen int) error {
	*title = strings.TrimSpace(*title)
	if *title == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if len(*title) > maxLen {
		return fmt.Errorf("%w: title must be %d characters or fewer", ErrInvalidInput, maxLen)
	}
	return nil
}

func validateSwatch(color string) error {
	if !IsSwatch(color) {
		return fmt.Errorf("%w: color must be a swatch (gray, blue, green, yellow, orange, pink, indigo)", ErrInvalidInput)
	}
	return nil
}

func validateDuration(mins int) error {
	if mins < minDurationMins || mins > maxDurationMins {
		return fmt.Errorf("%w: durationMins must be between %d and %d", ErrInvalidInput, minDurationMins, maxDurationMins)
	}
	return nil
}

func validateActionNotes(notes string) error {
	if len(notes) > maxActionNotesLen {
		return fmt.Errorf("%w: notes must be %d characters or fewer", ErrInvalidInput, maxActionNotesLen)
	}
	return nil
}

func (r *CreateProjectRequest) Validate() error {
	if err := validateTitle(&r.Title, maxProjectTitleLen); err != nil {
		return err
	}
	return validateSwatch(r.Color)
}

func (r *UpdateProjectRequest) Validate() error {
	if r.Title != nil {
		if err := validateTitle(r.Title, maxProjectTitleLen); err != nil {
			return err
		}
	}
	if r.Color != nil {
		if err := validateSwatch(*r.Color); err != nil {
			return err
		}
	}
	return nil
}

func (r *CreateActionRequest) Validate() error {
	if err := validateTitle(&r.Title, maxActionTitleLen); err != nil {
		return err
	}
	if err := validateActionNotes(r.Notes); err != nil {
		return err
	}
	if r.DurationMins != nil {
		return validateDuration(*r.DurationMins)
	}
	return nil
}

func (r *UpdateActionRequest) Validate() error {
	if r.Title != nil {
		if err := validateTitle(r.Title, maxActionTitleLen); err != nil {
			return err
		}
	}
	if r.Notes != nil {
		if err := validateActionNotes(*r.Notes); err != nil {
			return err
		}
	}
	if r.DurationMins.Set && r.DurationMins.Value != nil {
		return validateDuration(*r.DurationMins.Value)
	}
	return nil
}

func (r *ChangeStageRequest) Validate() error {
	if !r.Stage.IsBoardStage() {
		return fmt.Errorf("%w: stage must be one of braindump, refine, ready", ErrInvalidInput)
	}
	return nil
}

func (r *ReorderActionsRequest) Validate() error {
	if !r.Stage.IsBoardStage() {
		return fmt.Errorf("%w: only braindump, refine and ready can be reordered", ErrInvalidInput)
	}
	return nil
}

func (r *ScheduleActionRequest) Validate() error {
	if r.Date == nil || r.Date.IsZero() {
		return fmt.Errorf("%w: date is required", ErrInvalidInput)
	}
	return nil
}

func (r *MoveActionRequest) Validate() error {
	if r.Date == nil || r.Date.IsZero() {
		return fmt.Errorf("%w: date is required", ErrInvalidInput)
	}
	return nil
}

func (r *BreakDownActionRequest) Validate() error {
	if len(r.Titles) < minBreakDownTitles {
		return fmt.Errorf("%w: break down needs at least %d titles", ErrInvalidInput, minBreakDownTitles)
	}
	for i := range r.Titles {
		if err := validateTitle(&r.Titles[i], maxActionTitleLen); err != nil {
			return err
		}
	}
	return nil
}
