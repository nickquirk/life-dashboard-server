package domain

import (
	"log/slog"

	"gorm.io/gorm"
)

// AfterFind hydrates Goal from the flat goal_type/goal_target columns after every Find/Scan.
func (r *Routine) AfterFind(*gorm.DB) error {
	if r.GoalType != nil && r.GoalTarget != nil {
		r.Goal = &Goal{Type: GoalType(*r.GoalType), Target: *r.GoalTarget}
	}
	return nil
}

// RoutineWithStats embeds Routine, but GORM resolves AfterFind on the destination
// type; an explicit forwarder makes the behaviour unambiguous on Scan(&[]RoutineWithStats).
func (r *RoutineWithStats) AfterFind(tx *gorm.DB) error {
	slog.Info("RWS AfterFind", "id", r.ID, "goalType", r.GoalType, "goalTarget", r.GoalTarget)
	return r.Routine.AfterFind(tx)
}
