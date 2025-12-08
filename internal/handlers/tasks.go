package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/config"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

type TaskLists struct {
	Tasks []TaskList
}

type TaskList struct {
	Title string `json:"title"`
	Id    string `json:"id"`
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!\n"))
}

func (h *Handler) getTaskLists(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Retrieve UserID from context (set by Middleware)
	userID, ok := ctx.Value(UserIDKey).(uint)
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	// 2. Fetch User from DB to get access/refresh tokens
	userReq := domain.GetUserRequest{Id: userID}
	user, err := h.Service.GetUser(userReq)
	if err != nil {
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	// Build the Token Source dynamically
	tok := &oauth2.Token{
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
		Expiry:       user.TokenExpiry,
		TokenType:    "Bearer",
	}

	config := config.GetGoogleConfig()
	// This client handles token refreshing automatically!
	client := config.Client(ctx, tok)

	srv, err := tasks.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Printf("Unable to retrieve tasks Client: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	t, err := srv.Tasklists.List().Do()
	if err != nil {
		log.Printf("Unable to retrieve task lists: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t.Items) // Stream directly to response
}

func (h *Handler) getTasksInList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskListID := chi.URLParam(r, "taskListId")

	userID, ok := ctx.Value(UserIDKey).(uint)
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	userReq := domain.GetUserRequest{Id: userID}
	user, err := h.Service.GetUser(userReq)
	if err != nil {
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	// Build the Token Source dynamically
	tok := &oauth2.Token{
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
		Expiry:       user.TokenExpiry,
		TokenType:    "Bearer",
	}

	googleConfig := config.GetGoogleConfig()
	// This client handles token refreshing automatically
	client := googleConfig.Client(ctx, tok)

	srv, err := tasks.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Printf("Unable to retrieve tasks Client: %v", err)
		http.Error(w, "Service Error", http.StatusInternalServerError)
		return
	}

	// Call the Google API
	t, err := srv.Tasks.List(taskListID).Do()
	if err != nil {
		log.Printf("Unable to retrieve tasks: %v", err)
		http.Error(w, "Google API Error", http.StatusInternalServerError)
		return
	}

	// Set the content type of the response to application/json
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t.Items)
}
