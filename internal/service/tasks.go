package service

import (
	"context"
	"fmt"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/config"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

// Helper to get Google Client (moved from Handlers)
func (s *service) getGoogleClient(ctx context.Context, userID uint) (*tasks.Service, error) {
	// 1. Get User for tokens
	userResp, err := s.userRepo.Get(domain.GetUserRequest{Id: userID})
	if err != nil {
		return nil, err
	}

	// 2. Build Token
	tok := &oauth2.Token{
		AccessToken:  userResp.AccessToken,
		RefreshToken: userResp.RefreshToken,
		Expiry:       userResp.TokenExpiry,
		TokenType:    "Bearer",
	}

	// 3. Create Client
	conf := config.GetGoogleConfig()
	client := conf.Client(ctx, tok)

	return tasks.NewService(ctx, option.WithHTTPClient(client))
}

func (s *service) SyncAndGetTaskLists(ctx context.Context, userID uint) ([]domain.TaskList, error) {
	// 1. Connect to Google
	srv, err := s.getGoogleClient(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 2. Fetch from API
	gLists, err := srv.Tasklists.List().Do()
	if err != nil {
		return nil, err
	}

	// 3. Map to Domain Models
	var domainLists []domain.TaskList
	for _, item := range gLists.Items {
		domainLists = append(domainLists, domain.TaskList{
			ID:       item.Id,
			UserID:   userID,
			Title:    item.Title,
			Updated:  item.Updated,
			LastSync: time.Now(),
		})
	}

	// 4. Save to DB (Upsert)
	if err := s.taskRepo.UpsertTaskLists(domainLists); err != nil {
		return nil, err
	}

	// 5. Return from DB (Source of Truth)
	return s.taskRepo.GetTaskLists(userID)
}

// In internal/service/tasks.go

func (s *service) SyncAndGetAllTasks(ctx context.Context, userID uint, taskListID string) ([]domain.Task, error) {
	// 1. Connect to Google
	srv, err := s.getGoogleClient(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 2. Fetch ONLY active tasks (ShowCompleted = false)
	//    We want Google to tell us what is currently "Active"
	gTasks, err := srv.Tasks.List(taskListID).ShowCompleted(false).ShowHidden(false).Do()
	if err != nil {
		return nil, err
	}

	// 3. Map to Domain Models & Collect Active IDs
	var domainTasks []domain.Task
	var activeIDs []string

	for _, item := range gTasks.Items {
		// Just in case Google sends a stray completed one
		if item.Status != "needsAction" {
			continue
		}

		var dueDate *time.Time
		if item.Due != "" {
			parsed, err := time.Parse(time.RFC3339, item.Due)
			if err == nil {
				dueDate = &parsed
			}
		}

		domainTasks = append(domainTasks, domain.Task{
			ID:         item.Id,
			TaskListID: taskListID,
			Title:      item.Title,
			Status:     "needsAction", // Force status to active
			Notes:      item.Notes,
			Updated:    item.Updated,
			Due:        dueDate,
		})

		activeIDs = append(activeIDs, item.Id)
	}

	// 4. Upsert the Active Tasks (Updates existing ones, inserts new ones)
	if err := s.taskRepo.UpsertTasks(domainTasks); err != nil {
		return nil, err
	}

	// 5. THE MAGIC: Mark local tasks as 'completed' if they are missing from Google's active list
	if err := s.taskRepo.MarkTasksCompletedExcluding(taskListID, activeIDs); err != nil {
		fmt.Printf("Error marking tasks completed: %v\n", err)
		return nil, err
	}

	// 6. Update Sync Time
	s.taskRepo.UpdateListLastSync(taskListID, time.Now())

	// 7. Return ALL tasks (Active + Completed) so the UI can decide what to show
	return s.taskRepo.GetTasks(taskListID)
}

func (s *service) GetActiveTasks(ctx context.Context, taskListID string) ([]domain.Task, error) {
	// Just read from DB (Source of Truth)
	return s.taskRepo.GetActiveTasks(taskListID)
}
