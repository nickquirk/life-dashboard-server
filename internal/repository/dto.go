package repository

type CreateUserRequest struct {
	Email        string
	Picture      string
	AccessToken  string
	RefreshToken string
	TokenExpiry  string
}

type CreateUserResponse struct {
	Id uint
}
