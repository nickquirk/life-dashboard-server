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

type GetUserRequest struct {
	Id uint
}

type GetUserResponse struct {
	Email        string
	Picture      string
	AccessToken  string
	RefreshToken string
	TokenExpiry  time.Time
}
