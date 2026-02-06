package config

import (
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleConfig struct {
	GoogleLoginConfig oauth2.Config
}

var GoogleConfiguration GoogleConfig

func GetGoogleConfig() oauth2.Config {
	GoogleConfiguration.GoogleLoginConfig = oauth2.Config{
		RedirectURL:  os.Getenv("REDIRECT_URL"),
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes: []string{
			"https://www.googleapis.com/auth/tasks",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/calendar.readonly",
		},

		Endpoint: google.Endpoint,
	}
	return GoogleConfiguration.GoogleLoginConfig
}
