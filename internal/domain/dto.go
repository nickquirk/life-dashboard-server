package domain

type CreateUserRequest struct {
	Id           string `json:"id"`
	Email        string `json:"email"`
	Picture      string `json:"picture"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenExpiry  string `json:"token_expiry"`
}

type CreateUserResponse struct {
	Id uint
}
