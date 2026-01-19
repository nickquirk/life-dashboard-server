package domain

import "time"

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

type UpdateTaskRequest struct {
	// Local fields
	// Using pointers to distinguish between Update to 0 and don't update (nil)
	Quadrant     *int          `json:"quadrant"`
	DurationMins *int          `json:"durationMins"`
	Date         *NullableDate `json:"date"` // The "Planned" date

	// Google Fields
	Status *string    `json:"status"` // "needsAction" or "completed"
	Due    *time.Time `json:"due"`
}
