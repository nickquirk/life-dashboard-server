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
	ID       string    `gorm:"primaryKey;size:255" json:"id"` // Google's ID
	UserID   uint      `gorm:"index" json:"userId"`
	Title    string    `json:"title"`
	Updated  string    `json:"updated"`  // Google's timestamp
	LastSync time.Time `json:"lastSync"` // local timestamp of last successful sync

	// Relationships
	// Added omitempty so this field is ignored if not preloaded
	Tasks []Task `gorm:"foreignKey:TaskListID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"tasks,omitempty"`
	// Add timestamps for GORM to handle internal housekeeping
	CreatedAt time.Time
}

type Task struct {
	ID           string     `gorm:"primaryKey;size:255" json:"id"`          // Google's ID
	Parent       *string    `gorm:"index;size:255" json:"parent,omitempty"` // Google API usually calls this 'parent'
	TaskListID   string     `gorm:"index;size:255" json:"taskListId"`
	Title        string     `json:"title"`
	Status       string     `json:"status"` // "needsAction" or "completed"
	Due          *time.Time `json:"due,omitempty"`
	Notes        string     `json:"notes"`
	Updated      string     `json:"updated"` // Google's timestamp
	DurationMins int        `json:"durationMins"`
	Date         *time.Time `gorm:"type:datetime" json:"date,omitempty"`
	Subtasks     []Task     `gorm:"foreignKey:Parent" json:"subtasks,omitempty"` // So we can preload subtasks
	IsRepeating  bool       `json:"isRepeating"`
	Quadrant     int        `gorm:"default:0" json:"quadrant"`

	// Add timestamp for GORM to handle internal housekeeping
	CreatedAt time.Time
}

type CalendarEvent struct {
	ID           string    `gorm:"primaryKey;size:255" json:"id"` // Google's ID
	UserID       uint      `gorm:"index" json:"-"`
	Title        string    `json:"title"`
	Start        time.Time `gorm:"index" json:"start"` // Index for range queries
	End          time.Time `json:"end"`
	IsAllDay     bool      `json:"isAllDay"`
	CalendarName string    `json:"calendarName"`
	ColorID      string    `json:"colorId"`

	// Add timestamps for GORM to handle internal housekeeping
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type Zone struct {
	gorm.Model
	UserID     uint   `gorm:"index;not null" json:"id"` // Foreign key to User
	Label      string `json:"label"`
	StartTime  string `json:"startTime"`                         // Format: "HH:mm"
	EndTime    string `json:"endTime"`                           // Format: "HH:mm"
	Color      string `json:"color"`                             // e.g., "slate", "blue"
	DaysActive []uint `json:"daysActive" gorm:"serializer:json"` // 0 = Sunday, 1 = Monday, ..., 6 = Saturday
}
