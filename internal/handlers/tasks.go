package handlers

import (
	"context"
	"encoding/json"
	"fmt"
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
// Refactor to send string errors
func getTaskLists(w http.ResponseWriter, r *http.Request) {
	googleConfig := config.GoogleConfig()
	ctx := context.Background()

	client, err := utils.GetClient(&googleConfig)
	if err != nil {
		message := fmt.Sprintf("Unable to create client: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	srv, err := tasks.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		message := fmt.Sprintf("Unable to retrieve tasks client: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	t, err := srv.Tasklists.List().Do()
	if err != nil {
		message := fmt.Sprintf("Unable to retrieve tasklist: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	// clean this up by using the marshalJSON method?
	// Marshal the TaskLists struct into JSON
	marshalled, err := json.Marshal(t.Items)
	if err != nil {
		message := fmt.Sprintf("error marshalling tasklist items: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	// Unmarshal the JSON data into TaskLists struct
	var taskLists TaskLists
	if err := json.Unmarshal(marshalled, &taskLists.Tasks); err != nil {
		message := fmt.Sprintf("error unmarshalling JSON: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	// Set the content type of the response to application/json
	w.Header().Set("Content-Type", "application/json")

	resp, err := json.Marshal(taskLists)
	if err != nil {
		message := fmt.Sprintf("unable to marshal JSON: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
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
		message := fmt.Sprintf("Unable to retrieve tasklist: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	// Set the content type of the response to application/json
	w.Header().Set("Content-Type", "application/json")

	resp, err := t.MarshalJSON()
	if err != nil {
		message := fmt.Sprintf("unable to marshal JSON: %s", err)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	w.Write(resp)
}
