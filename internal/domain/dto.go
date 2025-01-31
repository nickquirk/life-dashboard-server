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
