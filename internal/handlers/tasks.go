package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/config"
	"github.com/nickquirk/life-dashboard-server/internal/utils"
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

// TODO
// Send errors instead of log.Fatal
func getTaskLists(w http.ResponseWriter, r *http.Request) {
	googleConfig := config.GoogleConfig()
	ctx := context.Background()

	client, err := utils.GetClient(&googleConfig)
	if err != nil {
		log.Fatalf("Unable to create client: %s\n", err)
	}

	srv, err := tasks.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve tasks Client %v", err)
	}

	t, err := srv.Tasklists.List().Do()
	if err != nil {
		log.Fatalf("Unable to retrieve task lists. %v", err)
	}

	// clean this up by using the marshalJSON method?
	// Marshal the TaskLists struct into JSON
	marshalled, err := json.Marshal(t.Items)
	if err != nil {
		log.Fatalf("Error marshalling TaskList items: %v", err)
	}

	// Unmarshal the JSON data into TaskLists struct
	var taskLists TaskLists
	if err := json.Unmarshal(marshalled, &taskLists.Tasks); err != nil {
		log.Fatalf("Error unmarshalling TaskLists: %v", err)
	}

	// Set the content type of the response to application/json
	w.Header().Set("Content-Type", "application/json")

	resp, err := json.Marshal(taskLists)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	w.Write(resp)
}

func getTasksInList(w http.ResponseWriter, r *http.Request) {
	googleConfig := config.GoogleConfig()
	ctx := context.Background()
	taskListParam := chi.URLParam(r, "taskListId")

	client, err := utils.GetClient(&googleConfig)
	if err != nil {
		message := fmt.Sprintf("Unable to create client: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	srv, err := tasks.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		message := fmt.Sprintf("Unable to retrieve tasks client: %v", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	t, err := srv.Tasks.List(taskListParam).Do()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set the content type of the response to application/json
	w.Header().Set("Content-Type", "application/json")

	resp, err := t.MarshalJSON()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Write(resp)
}
