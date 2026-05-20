package domain

import (
	"gorm.io/gorm"
)

// AfterFind hydrates Goal from the flat goal_type/goal_target columns after every Find/Scan.
func (r *Routine) AfterFind(*gorm.DB) error {
	if r.GoalType != nil && r.GoalTarget != nil {
		r.Goal = &Goal{Type: GoalType(*r.GoalType), Target: *r.GoalTarget}
	}
	return nil
}
