package domain

import "time"

// ------ User ---------
type CreateUserRequest struct {
	Email        string    `json:"email"`
	Picture      string    `json:"picture"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenExpiry  time.Time `json:"token_expiry"`
}

type CreateUserResponse struct {
	Id uint `json:"id"`
}

type GetUserRequest struct {
	Id    uint   `json:"id"`
	Email string `json:"email"`
}

type GetUserResponse struct {
	Email        string    `json:"email"`
	Picture      string    `json:"picture"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenExpiry  time.Time `json:"token_expiry"`
}

// ------ Task ---------
type CreateTaskRequest struct {
	Title      string `json:"title"`
	Parent     string `json:"parent,omitempty"`   // Optional: ID of the parent task
	PreviousID string `json:"previous,omitempty"` // Optional: ID of task to insert after (for ordering)
}

type UpdateTaskRequest struct {
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

// ------ Calendar ---------
type GetEventsRequest struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
