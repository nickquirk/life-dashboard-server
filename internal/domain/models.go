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
	ID         string `gorm:"primaryKey;size:255"` // Google's ID
	TaskListID string `gorm:"index;size:255"`
	Title      string
	Status     string // "needsAction" or "completed"
	Due        *time.Time
	Notes      string
	Updated    string // Google's timestamp

	// Dashboard Fields (Ready for the UI)
	// 0 = Unsorted (Inbox), 1 = Do (Urgent/Imp), 2 = Schedule (Imp/Not Urgent), etc.
	Quadrant int `gorm:"default:0"`
}
