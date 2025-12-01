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
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/utils"
	"golang.org/x/oauth2"
)

// TODO
// break out user data fetch into separte function
// state as random variable in cookie
// reauthorise in cookie?

func GoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := config.GoogleConfiguration.GoogleLoginConfig.AuthCodeURL(
		os.Getenv("STATE"),
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	state := params["state"][0]
	if state != os.Getenv("STATE") {
		http.Error(w, "Google Auth states do not match", http.StatusInternalServerError)
		log.Println("Google Auth states do not match")
		return
	}

	code := params["code"][0]
	googleConfig := config.GetGoogleConfig()
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to parse Google response body", http.StatusInternalServerError)
		log.Println("Error:", err)
		return
	}

	// Add oauth2 tokens and expiry to userData
	userData := domain.CreateUserRequest{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenExpiry:  token.Expiry,
	}

	err = json.Unmarshal(body, &userData)
	if err != nil {
		http.Error(w, "Failed to unmarshal user data", http.StatusInternalServerError)
		return
	}

	// Save user to DB
	user, err := h.CreateUser(userData)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to save user to database - %s", err)
		http.Error(w, errMessage, http.StatusInternalServerError)
		return
	}

	// save id and email in JWT
	tok, err := utils.GenerateToken(user.Id, userData.Email)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to generate JWT - %s", err)
		http.Error(w, errMessage, http.StatusInternalServerError)
		return
	}

	cookie := http.Cookie{
		Name:  "life-dashboard",
		Value: tok,
	}

	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
