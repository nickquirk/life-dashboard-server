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
	Sessions     []Session `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Session struct {
	ID              uint      `gorm:"primarykey"`
	UserID          uint      `gorm:"index;not null"`
	AppRefreshToken string    `gorm:"type:text;not null"` // The hashed refresh token
	ExpiresAt       time.Time `gorm:"type:datetime;not null"`
	DeviceInfo      string    `gorm:"type:text"` // e.g., "MacBook Safari"
}

type TaskList struct {
	ID       string     `gorm:"primaryKey;size:255" json:"id"` // Google's ID
	UserID   uint       `gorm:"index" json:"userId"`
	Title    string     `json:"title"`
	Updated  string     `json:"updated"`  // Google's timestamp
	LastSync *time.Time `json:"lastSync"` // local timestamp of last successful sync

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
	Subtasks     []Task     `gorm:"foreignKey:Parent;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"subtasks,omitempty"` // So we can preload subtasks
	Quadrant     int        `gorm:"default:0" json:"quadrant"`
	Position     int        `gorm:"not null;default:0" json:"position"` // Local sibling order under Parent

	// Add timestamp for GORM to handle internal housekeeping
	CreatedAt time.Time
}

type CalendarEvent struct {
	ID           string    `gorm:"primaryKey;size:255" json:"id"` // Google's ID
	UserID       uint      `gorm:"index:idx_ce_user_start,priority:1" json:"-"`
	Title        string    `json:"title"`
	Start        time.Time `gorm:"index:idx_ce_user_start,priority:2" json:"start"`
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
	UserID     uint   `gorm:"index;not null" json:"userId"` // Foreign key to User
	Label      string `json:"label"`
	StartTime  string `json:"startTime"`                         // Format: "HH:mm"
	EndTime    string `json:"endTime"`                           // Format: "HH:mm"
	Color      string `json:"color"`                             // e.g., "slate", "blue"
	DaysActive []uint `json:"daysActive" gorm:"serializer:json"` // 0 = Sunday, 1 = Monday, ..., 6 = Saturday
}

type Feedback struct {
	gorm.Model
	UserID  uint    `gorm:"index;not null" json:"userId"`
	Type    string  `json:"type"`    // e.g., "bug", "feature request"
	AppArea *string `json:"appArea"` // Which area of the app the feedback is about
	Message string  `json:"message"`
}

type Scratchpad struct {
	gorm.Model
	UserId  uint   `gorm:"uniqueIndex:idx_user_date;not null" json:"userId"`
	Date    string `gorm:"uniqueIndex:idx_user_date;not null;size:10" json:"date"` // Format: YYYY-MM-DD
	Content string `gorm:"type:text" json:"content"`
}

type GoalType string

const (
	GoalTypeTime  GoalType = "time"
	GoalTypeCount GoalType = "count"
)

type Goal struct {
	Type   GoalType `json:"type"`
	Target int      `json:"target"`
}

type Routine struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // "-" hides this from the frontend entirely

	UserID       uint   `gorm:"index;not null" json:"userId"`
	Title        string `json:"title"`
	DurationMins int    `json:"durationMins"`

	// Storage columns — hidden from JSON; clients see `goal` instead.
	GoalType   *string `gorm:"column:goal_type" json:"-"`
	GoalTarget *int    `gorm:"column:goal_target" json:"-"`
	// Transient assembled view, populated by AfterFind. Not persisted.
	Goal *Goal `gorm:"-" json:"goal,omitempty"`

	ResetPeriod *string `json:"resetPeriod,omitempty"` // NULL = one_off, "weekly", "monthly"
	IsArchived  bool    `gorm:"not null;default:false;index" json:"isArchived"`
}

type RoutineInstance struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID       uint      `gorm:"index:idx_ri_user_date,priority:1;not null" json:"userId"`
	RoutineID    uint      `gorm:"index;not null" json:"routineId"`
	Routine      Routine   `gorm:"foreignKey:RoutineID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"routine"` // Auto-fetches the template data
	Date         time.Time `gorm:"index:idx_ri_user_date,priority:2;type:datetime" json:"date"`
	Status       string    `json:"status"`                 // "needsAction" or "completed"
	DurationMins *int      `json:"durationMins,omitempty"` // Only populated if user overrides the template
	Label        string    `gorm:"size:200" json:"label"`  // Free-text per-instance annotation; "" = none
}

// Define supported types for notes
const (
	NoteTypeText      = "text"
	NoteTypeChecklist = "checklist"
	NoteTypeBullet    = "bullet"
)

type Note struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID     uint   `gorm:"index;not null" json:"userId"`
	Title      string `json:"title"`
	Type       string `gorm:"type:varchar(20);not null;default:'text'" json:"type"` // text, checklist, bullet
	Content    string `gorm:"type:text" json:"content"`                             // Used ONLY if Type == 'text'
	Color      string `json:"color"`
	IsPinned   bool   `gorm:"default:false" json:"isPinned"`
	IsArchived bool   `gorm:"default:false;index" json:"isArchived"`

	// Items are only populated if Type == 'checklist' or 'bullet'
	Items []NoteItem `gorm:"foreignKey:NoteID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"items,omitempty"`
}

type NoteItem struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	NoteID      uint   `gorm:"index;not null" json:"noteId"`
	Content     string `json:"content"`
	IsCompleted bool   `gorm:"default:false" json:"isCompleted"` // Ignored by FE if parent type is 'bullet'
	Position    int    `gorm:"not null;default:0" json:"position"`
}
