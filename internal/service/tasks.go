package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"golang.org/x/oauth2"
	"google.golang.org/api/tasks/v1"
	"gorm.io/gorm/clause"
)

// READ: Fast, Database only
func (s *service) GetTaskLists(userID uint) ([]domain.TaskList, error) {
	return s.taskRepo.GetTaskLists(userID)
}

// SYNC: Fetch from Google, Save to DB
func (s *service) SyncTaskLists(ctx context.Context, userID uint) error {
	srv, err := s.getGoogleTaskService(ctx, userID)
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

func (s *service) CreateTask(ctx context.Context, userID uint, taskListID string, req domain.CreateTaskRequest) (domain.Task, error) {
	srv, err := s.getGoogleTaskService(ctx, userID)
	if err != nil {
		return domain.Task{}, err
	}

	googleTask := &tasks.Task{
		Title: req.Title,
	}

	// Prepare the Insert call
	insertCall := srv.Tasks.Insert(taskListID, googleTask)

	// Use the .Parent() method to set the query parameter
	if req.Parent != "" {
		insertCall = insertCall.Parent(req.Parent)
	}

	// If it's a subtask, set the parent
	if req.Parent != "" {
		googleTask.Parent = req.Parent
	}

	// Call Google API
	createdGTask, err := insertCall.Do()
	if err != nil {
		return domain.Task{}, fmt.Errorf("google api insert failed: %w", err)
	}

	// Map response to Domain Model
	newTask := domain.Task{
		ID:         createdGTask.Id,
		TaskListID: taskListID,
		Title:      createdGTask.Title,
		Status:     createdGTask.Status, // Usually "needsAction"
		Updated:    createdGTask.Updated,
		Notes:      createdGTask.Notes,
	}

	// If Google returned it, use it. If not, use what we requested.
	if createdGTask.Parent != "" {
		parent := createdGTask.Parent
		newTask.Parent = &parent
	} else if req.Parent != "" {
		// Fallback: Google didn't echo it, but we know it belongs to this parent
		parent := req.Parent
		newTask.Parent = &parent
	}
	// Save to DB
	if err := s.taskRepo.CreateTask(newTask); err != nil {
		return domain.Task{}, err
	}

	return newTask, nil
}

func (s *service) GetTasks(ctx context.Context, taskListID string) ([]domain.Task, error) {
	// Directly return what we have in db. No Google API calls.
	return s.taskRepo.GetTasks(taskListID)
}

func (s *service) SyncTasks(ctx context.Context, userID uint, taskListID string) error {
	// Connect to Google
	srv, err := s.getGoogleTaskService(ctx, userID)
	if err != nil {
		return err
	}

	// Fetch ONLY active tasks (ShowCompleted = false)
	//    We want Google to tell us what is currently "Active"
	gTasks, err := srv.Tasks.List(taskListID).ShowCompleted(false).ShowHidden(true).Do()
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

		var parent *string
		if item.Parent != "" {
			// We capture the value in a new variable to get its address safely
			val := item.Parent
			parent = &val
		}

		domainTasks = append(domainTasks, domain.Task{
			ID:         item.Id,
			Parent:     parent,
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

func (s *service) UpdateTask(ctx context.Context, userID uint, taskID string, req domain.UpdateTaskRequest) error {
	task, err := s.taskRepo.GetTaskByID(taskID)
	if err != nil {
		return err
	}

	// DETECT MOVE: If TaskListID is present and different, trigger the Move logic
	if req.TaskListID != nil && *req.TaskListID != task.TaskListID {
		return s.moveTask(ctx, userID, task, *req.TaskListID, req)
	}

	updates := make(map[string]interface{})

	// Handle Standard Fields
	if req.Quadrant != nil {
		updates["quadrant"] = *req.Quadrant
	}
	if req.DurationMins != nil {
		updates["duration_mins"] = *req.DurationMins
	}

	// Handle Date/Time Logic, set or clear the date
	if req.Date != nil {
		if req.Date.Valid {
			updates["date"] = req.Date.Time
		} else {
			updates["date"] = nil
		}
	}

	// Handle Google Sync
	googleTask := &tasks.Task{}
	shouldPatch := false

	if req.Title != nil {
		googleTask.Title = *req.Title
		updates["title"] = *req.Title
		shouldPatch = true
	}
	if req.Notes != nil {
		googleTask.Notes = *req.Notes
		updates["notes"] = *req.Notes
		shouldPatch = true

		// Force send an empty string if the notes field has been cleared otherwise 'omitempty' will remove it
		if *req.Notes == "" {
			googleTask.ForceSendFields = append(googleTask.ForceSendFields, "Notes")
		}
	}

	if req.Status != nil {
		googleTask.Status = *req.Status
		updates["status"] = *req.Status
		shouldPatch = true
	}
	if req.Due != nil {
		googleTask.Due = req.Due.Format(time.RFC3339)
		updates["due"] = *req.Due
		shouldPatch = true
	}

	if shouldPatch {
		srv, err := s.getGoogleTaskService(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to get Google client: %w", err)
		}

		// Capture the response to get the official 'Updated' timestamp
		updatedGTask, err := srv.Tasks.Patch(task.TaskListID, taskID, googleTask).Do()
		if err != nil {
			return fmt.Errorf("failed to sync to Google: %w", err)
		}

		// Update the local timestamp to match Google's server time immediately
		updates["updated"] = updatedGTask.Updated
	}

	return s.taskRepo.UpdateTask(taskID, updates)
}

func (s *service) DeleteTask(ctx context.Context, userID uint, taskID string) error {
	// Get the task to find its taskListID
	task, err := s.taskRepo.GetTaskByID(taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	// Delete from Google Tasks API
	srv, err := s.getGoogleTaskService(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get Google client: %w", err)
	}

	err = srv.Tasks.Delete(task.TaskListID, taskID).Do()
	if err != nil {
		// Log but continue - task might already be deleted on Google's side
		fmt.Printf("Warning: Could not delete from Google Tasks: %v\n", err)
	}

	// Delete from local database
	if err := s.taskRepo.DeleteTask(taskID); err != nil {
		return fmt.Errorf("failed to delete task from database: %w", err)
	}

	return nil
}

func (s *service) isTokenError(err error) bool {
	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) {
		return true
	}
	return false
}

// Private helper to handle the "Delete Old + Create New" dance
func (s *service) moveTask(ctx context.Context, userID uint, currentTask domain.Task, newListID string, req domain.UpdateTaskRequest) error {
	srv, err := s.getGoogleTaskService(ctx, userID)
	if err != nil {
		return err
	}

	// Merge current data with any updates in the request
	title := currentTask.Title
	if req.Title != nil {
		title = *req.Title
	}

	notes := currentTask.Notes
	if req.Notes != nil {
		notes = *req.Notes
	}

	// Default to CURRENT status, not "needsAction"
	// This ensures that if you just move a task without changing status, it stays completed/active
	status := currentTask.Status
	if req.Status != nil {
		status = *req.Status
	}

	// Create Google Object
	googleTask := &tasks.Task{
		Title:  title,
		Notes:  notes,
		Status: status,
	}
	if req.Due != nil {
		googleTask.Due = req.Due.Format(time.RFC3339)
	}

	// Insert into NEW List on Google
	newGTask, err := srv.Tasks.Insert(newListID, googleTask).Do()
	if err != nil {
		return fmt.Errorf("failed to create task in new list: %w", err)
	}

	// Delete from OLD List on Google
	// We do this after successful insert to prevent data loss if insert fails
	_ = srv.Tasks.Delete(currentTask.TaskListID, currentTask.ID).Do()

	// Update Database (Atomic Swap)
	// We delete the old record and create a new one because the Primary Key (Google ID) has changed.

	// Prepare new domain task
	newTask := domain.Task{
		ID:         newGTask.Id,
		TaskListID: newListID,
		Title:      newGTask.Title,
		Status:     newGTask.Status,
		Updated:    newGTask.Updated,
		Notes:      newGTask.Notes,
		// Carry over local-only fields
		Quadrant:     currentTask.Quadrant,
		DurationMins: currentTask.DurationMins,
		IsRepeating:  currentTask.IsRepeating,
		Date:         currentTask.Date, // Default to old date
	}

	// Apply overrides for local fields from request
	if req.Quadrant != nil {
		newTask.Quadrant = *req.Quadrant
	}
	if req.Date != nil {
		if req.Date.Valid {
			newTask.Date = &req.Date.Time
		} else {
			newTask.Date = nil
		}
	}

	// Transaction to swap them
	tx := s.taskRepo.BeginTx()

	// Delete the old task ID
	if err := tx.Where("id = ?", currentTask.ID).Delete(&domain.Task{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Use Upsert (OnConflict) instead of Create
	// If 'newTask.ID' happens to collide with an existing record (zombie data/race condition),
	// this will overwrite it safely instead of crashing with UNIQUE constraint failed.
	if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&newTask).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
