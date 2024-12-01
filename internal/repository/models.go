package repository

type CreateUserRequest struct {
	Id           string
	Email        string
	Picture      string
	AccessToken  string
	RefreshToken string
	TokenExpiry  string
}

type CreateUserResponse struct {
	Id           string
	Email        string
	Picture      string
	AccessToken  string
	RefreshToken string
	TokenExpiry  string
}
