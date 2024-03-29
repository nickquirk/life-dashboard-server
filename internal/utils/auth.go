package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/models"
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

	expiryTime, err := time.Parse(time.RFC3339, user.TokenExpiry)
	if err != nil {
		return nil, err
	}

	tok.Expiry = expiryTime

	return config.Client(context.Background(), tok), nil
}

// TODO
// Add logic to check if user is current user or already exists
func SaveUserToFile(path string, user models.User) error {
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

func GetUserFromFile(file string) (models.User, error) {
	user := models.User{}
	f, err := os.Open(file)
	if err != nil {
		return user, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return user, err
	}
	err = json.Unmarshal(data, &user)
	if err != nil {
		return user, err
	}
	return user, nil
}
