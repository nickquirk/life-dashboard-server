package repository

import "time"

type CreateUserRequest struct {
	Email        string
	Picture      string
	AccessToken  string
	RefreshToken string
	TokenExpiry  time.Time
}

type CreateUserResponse struct {
	Id uint
}
