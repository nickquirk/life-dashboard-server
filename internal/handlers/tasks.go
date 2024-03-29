package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/nickquirk/life-dashboard-server/internal/config"
	"github.com/nickquirk/life-dashboard-server/internal/utils"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

func getTasks(w http.ResponseWriter, r *http.Request) {
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

	t, err := srv.Tasklists.List().MaxResults(10).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve task lists. %v", err)
	}

	list := fmt.Sprintln("Task Lists:")
	if len(t.Items) > 0 {
		for _, i := range t.Items {
			fmt.Printf("%s (%s)\n", i.Title, i.Id)
		}
	} else {
		fmt.Print("No task lists found.")
	}
	w.Write([]byte(list))
}
