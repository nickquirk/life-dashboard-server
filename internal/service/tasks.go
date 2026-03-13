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
func (s *service) GetTaskLists(req domain.GetTaskListsRequest) (domain.GetTaskListsResponse, error) {
	lists, err := s.taskRepo.GetTaskLists(req.UserID)
	if err != nil {
		return domain.GetTaskListsResponse{}, err
	}

	return domain.GetTaskListsResponse{
		TaskLists: lists,
	}, nil
}

// SYNC: Fetch from Google, Save to DB
func (s *service) SyncTaskLists(ctx context.Context, req domain.SyncTaskListsRequest) (domain.SyncTaskListsResponse, error) {
	userID := req.UserID

	srv, err := s.getGoogleTaskService(ctx, userID)
	if err != nil {
		return domain.SyncTaskListsResponse{}, err
	}

	// Fetch
	gLists, err := srv.Tasklists.List().Do()
	if err != nil {
		if s.isTokenError(err) {
			return domain.SyncTaskListsResponse{}, errors.New("unauthorized: refresh token invalid")
		}
		return domain.SyncTaskListsResponse{}, err
	}

	// Map
	var domainLists []domain.TaskList
	for _, item := range gLists.Items {
		domainLists = append(domainLists, domain.TaskList{
			ID:      item.Id,
			UserID:  userID,
			Title:   item.Title,
			Updated: item.Updated,
		})
	}

	// Save (Command only, returns error)
	if err := s.taskRepo.UpsertTaskLists(domainLists); err != nil {
		return domain.SyncTaskListsResponse{}, err
	}

	// Fetch the fresh lists from the DB and return them
	freshLists, err := s.taskRepo.GetTaskLists(userID)
	if err != nil {
		return domain.SyncTaskListsResponse{}, err
	}

	return domain.SyncTaskListsResponse{TaskLists: freshLists}, nil
}

func (s *service) CreateTask(ctx context.Context, req domain.CreateTaskRequest) (domain.CreateTaskResponse, error) {
	userID := req.UserID
	taskListID := req.TaskListID

	if err := s.taskRepo.VerifyTaskListOwner(userID, taskListID); err != nil {
		return domain.CreateTaskResponse{}, fmt.Errorf("task list not found: %w", err)
	}

	srv, err := s.getGoogleTaskService(ctx, userID)
	if err != nil {
		return domain.CreateTaskResponse{}, err
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
		return domain.CreateTaskResponse{}, fmt.Errorf("google api insert failed: %w", err)
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

	// Map local fields
	if req.Quadrant != nil {
		newTask.Quadrant = *req.Quadrant
	}
	if req.DurationMins != nil {
		newTask.DurationMins = *req.DurationMins
	}
	if req.Date != nil {
		newTask.Date = req.Date
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
		return domain.CreateTaskResponse{}, err
	}

	return domain.CreateTaskResponse{
		ID:           newTask.ID,
		Parent:       newTask.Parent,
		TaskListID:   newTask.TaskListID,
		Title:        newTask.Title,
		Status:       newTask.Status,
		Due:          newTask.Due,
		Notes:        newTask.Notes,
		Updated:      newTask.Updated,
		DurationMins: newTask.DurationMins,
		Date:         newTask.Date,
		Quadrant:     newTask.Quadrant,
	}, nil
}

func (s *service) GetTasks(ctx context.Context, req domain.GetTasksRequest) (domain.GetTasksResponse, error) {
	if err := s.taskRepo.VerifyTaskListOwner(req.UserID, req.TaskListID); err != nil {
		return domain.GetTasksResponse{}, fmt.Errorf("task list not found: %w", err)
	}

	tasks, err := s.taskRepo.GetTasks(req.TaskListID)
	if err != nil {
		return domain.GetTasksResponse{}, err
	}

	return domain.GetTasksResponse{
		Tasks: tasks,
	}, nil
}

func (s *service) SyncTasks(ctx context.Context, req domain.SyncTasksRequest) (domain.SyncTasksResponse, error) {
	userID := req.UserID
	taskListID := req.TaskListID

	if err := s.taskRepo.VerifyTaskListOwner(userID, taskListID); err != nil {
		return domain.SyncTasksResponse{}, fmt.Errorf("task list not found: %w", err)
	}

	// Connect to Google
	srv, err := s.getGoogleTaskService(ctx, userID)
	if err != nil {
		return domain.SyncTasksResponse{}, err
	}

	// Fetch ONLY active tasks (ShowCompleted = false)
	gTasks, err := srv.Tasks.List(taskListID).ShowCompleted(false).ShowHidden(true).Do()
	if err != nil {
		if s.isTokenError(err) {
			return domain.SyncTasksResponse{}, errors.New("unauthorized: refresh token invalid")
		}
		return domain.SyncTasksResponse{}, err
	}

	// Map to Domain Models & Collect Active IDs
	var domainTasks []domain.Task
	var activeIDs []string
	pendingMap := make(map[string]domain.Task) // NEW: Map for sorting

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
			val := item.Parent
			parent = &val
		}

		task := domain.Task{
			ID:         item.Id,
			Parent:     parent,
			TaskListID: taskListID,
			Title:      item.Title,
			Status:     "needsAction",
			Notes:      item.Notes,
			Updated:    item.Updated,
			Due:        dueDate,
		}

		domainTasks = append(domainTasks, task)
		pendingMap[task.ID] = task // Add to our map for sorting
		activeIDs = append(activeIDs, item.Id)
	}

	// Topological Sort (Parents First)
	var sortedTasks []domain.Task

	for len(pendingMap) > 0 {
		progress := false
		for id, t := range pendingMap {
			// A task is ready to be inserted if it has no parent,
			// OR if its parent is NO LONGER in the pending map (meaning it was already sorted or isn't in this batch)
			if t.Parent == nil {
				sortedTasks = append(sortedTasks, t)
				delete(pendingMap, id)
				progress = true
			} else if _, parentStillPending := pendingMap[*t.Parent]; !parentStillPending {
				sortedTasks = append(sortedTasks, t)
				delete(pendingMap, id)
				progress = true
			}
		}

		// Fallback for infinite loops (e.g. Google sends a child but the parent is completely missing)
		if !progress {
			for id, t := range pendingMap {
				// To strictly prevent FK errors on orphaned tasks, nullify the parent
				t.Parent = nil
				sortedTasks = append(sortedTasks, t)
				delete(pendingMap, id)
			}
		}
	}

	tx := s.taskRepo.BeginTx()
	if tx.Error != nil {
		return domain.SyncTasksResponse{}, tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create a specialized repo that uses this transaction
	txRepo := s.taskRepo.WithTx(tx)

	if err := txRepo.UpsertTasks(sortedTasks); err != nil {
		tx.Rollback()
		return domain.SyncTasksResponse{}, err
	}

	// Mark local tasks as 'completed' if they are missing from Google's active list
	if err := txRepo.MarkTasksCompletedExcluding(taskListID, activeIDs); err != nil {
		fmt.Printf("Error marking tasks completed: %v\n", err)
		tx.Rollback()
		return domain.SyncTasksResponse{}, err
	}

	// Update Sync Time
	if err := txRepo.UpdateListLastSync(taskListID, time.Now()); err != nil {
		tx.Rollback()
		return domain.SyncTasksResponse{}, err
	}

	if err := tx.Commit().Error; err != nil {
		return domain.SyncTasksResponse{}, err
	}

	// Fetch the fresh tasks from the DB and return them
	freshTasks, err := s.taskRepo.GetTasks(taskListID)
	if err != nil {
		return domain.SyncTasksResponse{}, err
	}

	return domain.SyncTasksResponse{Tasks: freshTasks}, nil
}

func (s *service) UpdateTask(ctx context.Context, req domain.UpdateTaskRequest) (domain.UpdateTaskResponse, error) {
	userID := req.UserID
	taskID := req.TaskID

	taskListID, err := s.taskRepo.GetTaskListIDForTask(taskID)
	if err != nil {
		return domain.UpdateTaskResponse{}, err
	}

	// Verify the task belongs to this user (via its task list)
	if err := s.taskRepo.VerifyTaskListOwner(userID, taskListID); err != nil {
		return domain.UpdateTaskResponse{}, fmt.Errorf("task not found: %w", err)
	}

	// DETECT MOVE: If TaskListID is present and different, trigger the Move logic
	if req.TaskListID != nil && *req.TaskListID != taskListID {
		task, err := s.taskRepo.GetTaskByID(taskID)
		if err != nil {
			return domain.UpdateTaskResponse{}, err
		}
		if err := s.moveTask(ctx, userID, task, *req.TaskListID, req); err != nil {
			return domain.UpdateTaskResponse{}, err
		}
		return domain.UpdateTaskResponse{ID: taskID}, nil
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
			return domain.UpdateTaskResponse{}, fmt.Errorf("failed to get Google client: %w", err)
		}

		// Capture the response to get the official 'Updated' timestamp
		updatedGTask, err := srv.Tasks.Patch(taskListID, taskID, googleTask).Do()
		if err != nil {
			return domain.UpdateTaskResponse{}, fmt.Errorf("failed to sync to Google: %w", err)
		}

		// Update the local timestamp to match Google's server time immediately
		updates["updated"] = updatedGTask.Updated
	}

	if err := s.taskRepo.UpdateTask(taskID, updates); err != nil {
		return domain.UpdateTaskResponse{}, err
	}

	return domain.UpdateTaskResponse{ID: taskID}, nil
}

func (s *service) DeleteTask(ctx context.Context, req domain.DeleteTaskRequest) (domain.DeleteTaskResponse, error) {
	userID := req.UserID
	taskID := req.TaskID

	// Get just the task list ID for authorization
	taskListID, err := s.taskRepo.GetTaskListIDForTask(taskID)
	if err != nil {
		return domain.DeleteTaskResponse{}, fmt.Errorf("task not found: %w", err)
	}

	// Verify the task belongs to this user (via its task list)
	if err := s.taskRepo.VerifyTaskListOwner(userID, taskListID); err != nil {
		return domain.DeleteTaskResponse{}, fmt.Errorf("task not found: %w", err)
	}

	// Delete from Google Tasks API
	srv, err := s.getGoogleTaskService(ctx, userID)
	if err != nil {
		return domain.DeleteTaskResponse{}, fmt.Errorf("failed to get Google client: %w", err)
	}

	err = srv.Tasks.Delete(taskListID, taskID).Do()
	if err != nil {
		// Log but continue - task might already be deleted on Google's side
		fmt.Printf("Warning: Could not delete from Google Tasks: %v\n", err)
	}

	// Delete from local database
	if err := s.taskRepo.DeleteTask(taskID); err != nil {
		return domain.DeleteTaskResponse{}, fmt.Errorf("failed to delete task from database: %w", err)
	}

	return domain.DeleteTaskResponse{ID: taskID}, nil
}

func (s *service) DeleteTasks(ctx context.Context, req domain.DeleteTasksRequest) (domain.DeleteTasksResponse, error) {
	userID := req.UserID

	if len(req.TaskIDs) == 0 {
		return domain.DeleteTasksResponse{IDs: []string{}}, nil
	}

	// Group tasks by task list for ownership verification
	taskListMap := make(map[string][]string) // taskListID -> []taskID
	for _, taskID := range req.TaskIDs {
		taskListID, err := s.taskRepo.GetTaskListIDForTask(taskID)
		if err != nil {
			return domain.DeleteTasksResponse{}, fmt.Errorf("task not found: %s: %w", taskID, err)
		}
		taskListMap[taskListID] = append(taskListMap[taskListID], taskID)
	}

	// Verify ownership for each task list
	for taskListID := range taskListMap {
		if err := s.taskRepo.VerifyTaskListOwner(userID, taskListID); err != nil {
			return domain.DeleteTasksResponse{}, fmt.Errorf("task not found: %w", err)
		}
	}

	// Delete from Google Tasks API
	srv, err := s.getGoogleTaskService(ctx, userID)
	if err != nil {
		return domain.DeleteTasksResponse{}, fmt.Errorf("failed to get Google client: %w", err)
	}

	for taskListID, taskIDs := range taskListMap {
		for _, taskID := range taskIDs {
			err := srv.Tasks.Delete(taskListID, taskID).Do()
			if err != nil {
				// Log but continue - task might already be deleted on Google's side
				fmt.Printf("Warning: Could not delete task %s from Google Tasks: %v\n", taskID, err)
			}
		}
	}

	// Delete from local database
	if err := s.taskRepo.DeleteTasks(req.TaskIDs); err != nil {
		return domain.DeleteTasksResponse{}, fmt.Errorf("failed to delete tasks from database: %w", err)
	}

	return domain.DeleteTasksResponse{IDs: req.TaskIDs}, nil
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
	// Verify the user owns the destination list
	if err := s.taskRepo.VerifyTaskListOwner(userID, newListID); err != nil {
		return fmt.Errorf("destination task list not found: %w", err)
	}

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
