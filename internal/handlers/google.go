package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/nickquirk/life-dashboard-server/internal/config"
)

type User struct {
	Id      string `json:"id"`
	Email   string `json:"email"`
	Picture string `json:"picture"`
}

// TODO
// break out user data fetch into separte function
// state as random variable in cookie
// reauthorise in cookie?

func GoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := config.AppConfig.GoogleLoginConfig.AuthCodeURL(os.Getenv("STATE"))
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func GoogleCallback(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	state := params["state"][0]
	if state != os.Getenv("STATE") {
		http.Error(w, "Google Auth states do not match", http.StatusInternalServerError)
		log.Println("Google Auth states do not match")
		return
	}

	code := params["code"][0]
	googleConfig := config.GoogleConfig()
	ctx := context.Background()

	token, err := googleConfig.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "Code-Token exchange failed", http.StatusInternalServerError)
		log.Println("Error: ", err)
		return
	}

	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		http.Error(w, "User data fetch failed", http.StatusInternalServerError)
		log.Println("Error: ", err)
		return
	}

	userData := User{}

	rawUserData, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to parse Google response body", http.StatusInternalServerError)
		log.Println("Error:", err)
		return
	}

	err = json.Unmarshal(rawUserData, &userData)
	if err != nil {
		http.Error(w, "Failed to unmarshal user data", http.StatusInternalServerError)
		return
	}

	fmt.Printf("userData: %v/n", userData)
	http.Redirect(w, r, "/tasks", http.StatusTemporaryRedirect)
}
