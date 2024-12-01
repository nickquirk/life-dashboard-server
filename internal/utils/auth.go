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
	}

	// expiryLayout := "2006-01-02 15:04:05.999999999 -0700 MST m=+3669.687948501"
	// expiryTime, err := time.Parse(expiryLayout, user.TokenExpiry)
	// if err != nil {
	// 	return nil, err
	// }

	tok.Expiry = user.TokenExpiry

	return config.Client(context.Background(), tok), nil
}

// TODO
// Add logic to check if user is current user or already exists
func SaveUserToFile(path string, user domain.User) error {
	fmt.Printf("Saving user file to: %s\n", path)
	jsonData, err := json.Marshal(user)
	if err != nil {
		return err
	}
	// Write JSON data to file
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(jsonData)
	if err != nil {
		return err
	}
	return nil
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
