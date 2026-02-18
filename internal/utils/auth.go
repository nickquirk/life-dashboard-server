package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"golang.org/x/oauth2"
)

// Retrieve a token, saves the token, then returns the generated client.
func GetClient(config *oauth2.Config) (*http.Client, error) {
	// The file user.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	user, err := GetUserFromFile("user.json")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil, err
	}

	tok := &oauth2.Token{
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
		Expiry:       user.TokenExpiry,
	}

	return config.Client(context.Background(), tok), nil
}

// Retrieve user data from a JSON file
func GetUserFromFile(file string) (domain.User, error) {
	user := domain.User{}
	f, err := os.Open(file)
	if err != nil {
		return user, err
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&user)
	if err != nil {
		return user, err
	}
	return user, nil
}
