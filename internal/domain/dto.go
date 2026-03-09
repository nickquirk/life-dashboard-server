package domain

import (
	"time"
)

// ------ User ---------

type CreateUserRequest struct {
	Email        string    `json:"email"`
	Picture      string    `json:"picture"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenExpiry  time.Time `json:"token_expiry"`
}

type CreateUserResponse struct {
	ID uint `json:"id"`
}

type GetUserRequest struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
}

type GetUserResponse struct {
	Email        string    `json:"email"`
	Picture      string    `json:"picture"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenExpiry  time.Time `json:"token_expiry"`
}

type GetCurrentUserIDResponse struct {
	ID uint `json:"id"`
}

type UserProfileResponse struct {
	Email   string `json:"email"`
	Picture string `json:"picture"`
}

// ------ Task ---------

type GetTaskListsRequest struct {
	UserID uint `json:"-"`
}

type GetTaskListsResponse struct {
	TaskLists []TaskList `json:"taskLists"`
}

type SyncTaskListsRequest struct {
	UserID uint `json:"-"`
}

type SyncTaskListsResponse struct {
	TaskLists []TaskList `json:"taskLists"`
}

type CreateTaskRequest struct {
	UserID       uint       `json:"-"`
	TaskListID   string     `json:"-"`
	Title        string     `json:"title"`
	Parent       string     `json:"parent,omitempty"`   // Optional: ID of the parent task
	PreviousID   string     `json:"previous,omitempty"` // Optional: ID of task to insert after (for ordering)
	IsRepeating  *bool      `json:"isRepeating,omitempty"`
	Quadrant     *int       `json:"quadrant,omitempty"`
	DurationMins *int       `json:"durationMins,omitempty"`
	Date         *time.Time `json:"date,omitempty"`
}

type CreateTaskResponse struct {
	ID           string     `json:"id"`
	Parent       *string    `json:"parent,omitempty"`
	TaskListID   string     `json:"taskListId"`
	Title        string     `json:"title"`
	Status       string     `json:"status"`
	Due          *time.Time `json:"due,omitempty"`
	Notes        string     `json:"notes"`
	Updated      string     `json:"updated"`
	DurationMins int        `json:"durationMins"`
	Date         *time.Time `json:"date,omitempty"`
	Subtasks     []Task     `json:"subtasks,omitempty"`
	IsRepeating  bool       `json:"isRepeating"`
	Quadrant     int        `json:"quadrant"`
}

type GetTasksRequest struct {
	UserID     uint   `json:"-"`
	TaskListID string `json:"-"`
}

type GetTasksResponse struct {
	Tasks []Task `json:"tasks"`
}

type SyncTasksRequest struct {
	UserID     uint   `json:"-"`
	TaskListID string `json:"-"`
}

type SyncTasksResponse struct {
	Tasks []Task `json:"tasks"`
}

type UpdateTaskRequest struct {
	UserID uint   `json:"-"`
	TaskID string `json:"-"`
	// Local fields
	// Using pointers to distinguish between Update to 0 and don't update (nil)
	Title        *string       `json:"title"`
	Quadrant     *int          `json:"quadrant"`
	DurationMins *int          `json:"durationMins"`
	Date         *NullableDate `json:"date"` // The "Planned" date
	Notes        *string       `json:"notes"`
	TaskListID   *string       `json:"taskListId"`
	// Google Fields
	Status *string    `json:"status"` // "needsAction" or "completed"
	Due    *time.Time `json:"due"`
}

type UpdateTaskResponse struct {
	ID string `json:"id"`
}

type DeleteTaskRequest struct {
	UserID uint   `json:"-"`
	TaskID string `json:"-"`
}

type DeleteTaskResponse struct {
	ID string `json:"id"`
}

// ------ Calendar ---------

type GetCalendarEventsRequest struct {
	UserID uint      `json:"userId"`
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
}

type GetCalendarEventsResponse struct {
	Events []CalendarEvent `json:"events"`
}

// ------ Zone ---------

type ZoneResponse struct {
	ID         uint   `json:"id"`
	Label      string `json:"label"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
	Color      string `json:"color"`
	DaysActive []uint `json:"daysActive"`
}

type GetZonesRequest struct {
	UserID uint `json:"userId"`
}

type GetZonesResponse struct {
	Zones []ZoneResponse `json:"zones"`
}

type CreateZoneRequest struct {
	UserID     uint   `json:"userId"`
	Label      string `json:"label"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
	Color      string `json:"color"`
	DaysActive []uint `json:"daysActive"`
}

type CreateZoneResponse struct {
	ID uint `json:"id"`
}

type UpdateZoneRequest struct {
	ID         uint    `json:"-"`
	UserID     uint    `json:"-"`
	Label      *string `json:"label"`
	StartTime  *string `json:"startTime"`
	EndTime    *string `json:"endTime"`
	Color      *string `json:"color"`
	DaysActive []uint  `json:"daysActive"`
}

// Could return 204 No Content?
type UpdateZoneResponse struct {
	ID         uint      `json:"id"`
	UserID     uint      `json:"userId"`
	Label      string    `json:"label"`
	StartTime  string    `json:"startTime"`
	EndTime    string    `json:"endTime"`
	Color      string    `json:"color"`
	DaysActive []uint    `json:"daysActive"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type DeleteZoneRequest struct {
	ID     uint `json:"id"`
	UserID uint `json:"userId"`
}

type DeleteZoneResponse struct {
	ID uint `json:"id"`
}

// ------ Feedback ---------

type CreateFeedbackRequest struct {
	UserID  uint    `json:"userId"`
	Type    string  `json:"type"`              // e.g., "bug", "feature request"
	AppArea *string `json:"appArea,omitempty"` // e.g "auth", "drag and drop"
	Message string  `json:"message"`
}

type CreateFeedbackResponse struct {
	ID uint `json:"id"`
}

// ------ Scratchpad ---------

type GetScratchpadRequest struct {
	UserID uint   `json:"-"`
	Date   string `json:"date"`
}

type GetScratchpadResponse struct {
	Content string `json:"content"`
}

type UpsertScratchpadRequest struct {
	UserID  uint   `json:"-"`
	Date    string `json:"date"`
	Content string `json:"content"`
}

type UpsertScratchpadResponse struct {
	Content string `json:"content"`
}
