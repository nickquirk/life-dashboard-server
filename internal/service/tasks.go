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

		var parentID *string
		if item.Parent != "" {
			// We capture the value in a new variable to get its address safely
			val := item.Parent
			parentID = &val
		}

		domainTasks = append(domainTasks, domain.Task{
			ID:         item.Id,
			ParentID:   parentID,
			TaskListID: taskListID,
			Title:      item.Title,
			Status:     "needsAction", // Force status to active
			Notes:      item.Notes,
			Updated:    item.Updated,
			Due:        dueDate,
		})

		activeIDs = append(activeIDs, item.Id)
	}

	// Upsert the Active Tasks (Updates existing ones, inserts new ones)
	if err := s.taskRepo.UpsertTasks(domainTasks); err != nil {
		return nil, err
	}

	// Mark local tasks as 'completed' if they are missing from Google's active list
	if err := s.taskRepo.MarkTasksCompletedExcluding(taskListID, activeIDs); err != nil {
		fmt.Printf("Error marking tasks completed: %v\n", err)
		return nil, err
	}

	// Update Sync Time
	s.taskRepo.UpdateListLastSync(taskListID, time.Now())

	// Return ALL tasks (Active + Completed) so the UI can decide what to show
	return s.taskRepo.GetTasks(taskListID)
}

func (s *service) GetActiveTasks(ctx context.Context, taskListID string) ([]domain.Task, error) {
	// Just read from DB (Source of Truth)
	return s.taskRepo.GetActiveTasks(taskListID)
}

func (s *service) UpdateTask(ctx context.Context, userID uint, taskID string, req domain.UpdateTaskRequest) error {
	// Fetch task
	task, err := s.taskRepo.GetTaskByID(taskID)
	if err != nil {
		return err
	}

	updates := make(map[string]interface{})

	if req.Quadrant != nil {
		updates["quadrant"] = *req.Quadrant
	}
	if req.DurationMins != nil {
		updates["duration_mins"] = *req.DurationMins
	}
	if req.Date != nil {
		// Parse "2006-01-02" format
		parsedDate, err := time.Parse("2006-01-02", *req.Date)
		if err == nil {
			updates["date"] = parsedDate
		} else {
			// Log error or ignore if format is bad
			fmt.Printf("Error parsing date: %v\n", err)
		}
	}
	if req.ScheduledTime != nil {
		updates["scheduled_time"] = *req.ScheduledTime
	}
	if req.ScheduledMinute != nil {
		updates["scheduled_minute"] = *req.ScheduledMinute
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Due != nil {
		updates["due"] = *req.Due
	}

	// Handle Google sync (if status or due changed)
	if req.Status != nil || req.Due != nil {
		srv, err := s.getGoogleClient(ctx, userID)
		if err != nil {
			// Create a minimal Google Task object for patching
			googleTask := &tasks.Task{}
			shouldCallGoogle := false

			if req.Status != nil {
				googleTask.Status = *req.Status
				shouldCallGoogle = true
			}
			// Only sync due date if explicitly requested
			if req.Due != nil {
				googleTask.Due = req.Due.Format(time.RFC3339)
				shouldCallGoogle = true
			}

			if shouldCallGoogle {
				// ignore error so that db is still synced
				_, _ = srv.Tasks.Patch(task.TaskListID, taskID, googleTask).Do()
			}
		}
	}
	// Save to loacal DB
	return s.taskRepo.UpdateTask(taskID, updates)
}
