package service

import (
	"context"
	"errors"
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

// READ: Fast, Database only
func (s *service) GetTaskLists(userID uint) ([]domain.TaskList, error) {
	return s.taskRepo.GetTaskLists(userID)
}

// SYNC: Fetch from Google, Save to DB
func (s *service) SyncTaskLists(ctx context.Context, userID uint) error {
	// Connect
	srv, err := s.getGoogleClient(ctx, userID)
	if err != nil {
		return err
	}

	// Fetch
	gLists, err := srv.Tasklists.List().Do()
	if err != nil {
		if s.isTokenError(err) {
			return errors.New("unauthorized: refresh token invalid")
		}
		return err
	}

	// Map
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

	// Save (Command only, returns error)
	return s.taskRepo.UpsertTaskLists(domainLists)
}

func (s *service) GetTasks(ctx context.Context, taskListID string) ([]domain.Task, error) {
	// Directly return what we have in db. No Google API calls.
	return s.taskRepo.GetTasks(taskListID)
}

func (s *service) SyncTasks(ctx context.Context, userID uint, taskListID string) error {
	// Connect to Google
	srv, err := s.getGoogleClient(ctx, userID)
	if err != nil {
		return err
	}

	// Fetch ONLY active tasks (ShowCompleted = false)
	//    We want Google to tell us what is currently "Active"
	gTasks, err := srv.Tasks.List(taskListID).ShowCompleted(false).ShowHidden(false).Do()
	if err != nil {
		if s.isTokenError(err) {
			// TODO Clear invalid tokens in the db
			//s.userRepo.ClearTokens(userID)
			return errors.New("unauthorized: refresh token invalid")
		}
		return err
	}

	// Map to Domain Models & Collect Active IDs
	var domainTasks []domain.Task
	var activeIDs []string

	for _, item := range gTasks.Items {
		// Catch if Google sends a stray completed one
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

	tx := s.taskRepo.BeginTx()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create a specialized repo that uses this transaction
	txRepo := s.taskRepo.WithTx(tx)

	// Upsert the Active Tasks (Updates existing ones, inserts new ones)
	if err := txRepo.UpsertTasks(domainTasks); err != nil {
		tx.Rollback()
		return err
	}

	// Mark local tasks as 'completed' if they are missing from Google's active list
	if err := txRepo.MarkTasksCompletedExcluding(taskListID, activeIDs); err != nil {
		fmt.Printf("Error marking tasks completed: %v\n", err)
		tx.Rollback()
		return err
	}

	// Update Sync Time
	if err := txRepo.UpdateListLastSync(taskListID, time.Now()); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *service) GetActiveTasks(ctx context.Context, taskListID string) ([]domain.Task, error) {
	// Just read from DB (Source of Truth)
	return s.taskRepo.GetActiveTasks(taskListID)
}

func (s *service) UpdateTask(ctx context.Context, userID uint, taskID string, req domain.UpdateTaskRequest) error {
	task, err := s.taskRepo.GetTaskByID(taskID)
	if err != nil {
		return err
	}

	updates := make(map[string]interface{})

	// 1. Handle Standard Fields
	if req.Quadrant != nil {
		updates["quadrant"] = *req.Quadrant
	}
	if req.DurationMins != nil {
		updates["duration_mins"] = *req.DurationMins
	}

	// If nil, the field was not sent in JSON -> Do nothing
	if req.Date != nil {
		if req.Date.Valid {
			// Valid date provided -> Update it
			updates["date"] = req.Date.Time
		} else {
			// Sent as "" or null -> Clear it in DB
			updates["date"] = nil
		}
	}

	// ... Handle ScheduledTime logic ...

	// Handle Google Sync (Fixed Logic)
	if req.Status != nil || req.Due != nil {
		// Prepare the Patch
		googleTask := &tasks.Task{}
		shouldPatch := false

		if req.Status != nil {
			googleTask.Status = *req.Status
			updates["status"] = *req.Status // Update local map
			shouldPatch = true
		}
		if req.Due != nil {
			googleTask.Due = req.Due.Format(time.RFC3339)
			updates["due"] = *req.Due // Update local map
			shouldPatch = true
		}

		if shouldPatch {
			// Get Client
			srv, err := s.getGoogleClient(ctx, userID)
			if err != nil {
				// Decide: Fail request? Or Log and continue local only?
				// Usually, if sync fails, we might want to warn the user,
				// but let's assume we log and continue for now.
				fmt.Printf("Warning: Could not sync to Google: %v\n", err)
			} else {
				// ONLY Call Google if srv is valid
				_, err = srv.Tasks.Patch(task.TaskListID, taskID, googleTask).Do()
				if err != nil {
					fmt.Printf("Error patching Google Task: %v\n", err)
					// Optional: Return error here if you want strict consistency
				}
			}
		}
	}

	// 4. Save to Local DB
	return s.taskRepo.UpdateTask(taskID, updates)
}

func (s *service) isTokenError(err error) bool {
	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) {
		return true
	}
	return false
}
