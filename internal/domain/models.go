package domain

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email        string    `gorm:"unique;not null;size:255"`
	Picture      string    `gorm:"type:text"`
	AccessToken  string    `gorm:"type:text"`
	RefreshToken string    `gorm:"type:text"`
	TokenExpiry  time.Time `gorm:"type:datetime"`
}

type TaskList struct {
	ID       string `gorm:"primaryKey;size:255"` // Google's ID
	UserID   uint   `gorm:"index"`
	Title    string
	Updated  string    // Google's timestamp
	LastSync time.Time // local timestamp of last successful sync

	// Relationships
	Tasks []Task `gorm:"foreignKey:TaskListID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Task struct {
	ID              string  `gorm:"primaryKey;size:255"` // Google's ID
	ParentID        *string `gorm:"index;size:255"`      // Pointer allows nil (top-level tasks)
	TaskListID      string  `gorm:"index;size:255"`
	Title           string
	Status          string // "needsAction" or "completed"
	Due             *time.Time
	Notes           string
	Updated         string // Google's timestamp
	DurationMins    int    // Task duration
	ScheduledTime   int
	ScheduledMinute int
	Date            *time.Time
	Description     string
	Subtasks        []Task `gorm:"foreignKey:ParentID"` // So we can preload subtasks
	IsRepeating     bool
	Quadrant        int `gorm:"default:0"`
}
