package handlers

type CreateUserRequestDto struct {
	Email        string `json:"email"`
	Picture      string `json:"picture"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenExpiry  string `json:"token_expiry"`
}

type CreateUserResponseDto struct {
	Id uint `json:"id"`
}
