package handlers

import (
	"context"
	"net/http"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"golang.org/x/oauth2"
)

// TODO
// read session cookie and authenticate if valid

func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// cookie, err := r.Cookie("lifeDashboard")
		// if err != nil {
		// 	switch {
		// 	case errors.Is(err, http.ErrNoCookie):
		// 		http.Error(w, "cookie not found", http.StatusBadRequest)
		// 	default:
		// 		log.Println(err)
		// 		http.Error(w, "server error", http.StatusInternalServerError)
		// 	}
		// 	return
		// }

		// authToken, err := utils.VerifyToken(cookie.Value)
		// if err != nil {
		// 	http.Error(w, "not authorised", http.StatusUnauthorized)
		// 	fmt.Printf("error: %v\n", err)
		// 	return
		// }

		// fmt.Printf("auth tok: %s\n", authToken)

		//utils.SaveToken("token.json", authToken)

		//w.Write([]byte("Auth middleware hit!\n"))
		next.ServeHTTP(w, r)
	})
}

// Retrieve a token, saves the token, then returns the generated client.
func (h *Handler) GetClient(config *oauth2.Config) (*http.Client, error) {
	// The file user.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.

	//TODO
	// get id/email of user
	// get user from db

	userToLogIn := domain.GetUserRequest{}

	user, err := h.GetUser(userToLogIn)
	if err != nil {
		return config.Client(context.Background(), &oauth2.Token{}), err
	}

	tok := &oauth2.Token{
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
		Expiry:       user.TokenExpiry,
	}

	return config.Client(context.Background(), tok), nil
}
