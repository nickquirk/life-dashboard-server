package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/nickquirk/life-dashboard-server/internal/config"
	"github.com/nickquirk/life-dashboard-server/internal/models"
	"github.com/nickquirk/life-dashboard-server/internal/utils"
)

// TODO
// break out user data fetch into separte function
// state as random variable in cookie
// reauthorise in cookie?
// User in models/user

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

	rawUserData, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to parse Google response body", http.StatusInternalServerError)
		log.Println("Error:", err)
		return
	}

	userData := models.User{}
	userData.AccessToken = token.AccessToken

	err = json.Unmarshal(rawUserData, &userData)
	if err != nil {
		http.Error(w, "Failed to unmarshal user data", http.StatusInternalServerError)
		return
	}

	jwt, err := utils.GenerateToken(userData)
	if err != nil {
		http.Error(w, "failed to generate jwt", http.StatusInternalServerError)
		return
	}

	cookie := http.Cookie{
		Name:  "lifeDashboard",
		Value: jwt,
	}

	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
