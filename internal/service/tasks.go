package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

	gLists, err := srv.Tasklists.List().Do()
	if err != nil {
		if s.isTokenError(err) {
			return errors.New("unauthorized: refresh token invalid")
		}
		return err
	}

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

	return s.taskRepo.UpsertTaskLists(domainLists)
}

func (s *service) CreateTask(ctx context.Context, userID uint, taskListID string, req domain.CreateTaskRequest) (domain.Task, error) {
	if err := s.taskRepo.VerifyTaskListOwner(userID, taskListID); err != nil {
		return domain.Task{}, fmt.Errorf("task list not found: %w", err)
	}

	srv, err := s.getGoogleTaskService(ctx, userID)
	if err != nil {
		return domain.Task{}, err
	}

	googleTask := &tasks.Task{
		Title: req.Title,
	}

	insertCall := srv.Tasks.Insert(taskListID, googleTask)

	if req.Parent != "" {
		insertCall = insertCall.Parent(req.Parent)
	}

	if req.Parent != "" {
		googleTask.Parent = req.Parent
	}

	createdGTask, err := insertCall.Do()
	if err != nil {
		return domain.Task{}, fmt.Errorf("google api insert failed: %w", err)
	}

	newTask := domain.Task{
		ID:         createdGTask.Id,
		TaskListID: taskListID,
		Title:      createdGTask.Title,
		Status:     createdGTask.Status,
		Updated:    createdGTask.Updated,
		Notes:      createdGTask.Notes,
	}

	if createdGTask.Parent != "" {
		parent := createdGTask.Parent
		newTask.Parent = &parent
	} else if req.Parent != "" {
		parent := req.Parent
		newTask.Parent = &parent
	}

	if err := s.taskRepo.CreateTask(newTask); err != nil {
		return domain.Task{}, err
	}

	return newTask, nil
}

func (s *service) GetTasks(ctx context.Context, userID uint, taskListID string) ([]domain.Task, error) {
	if err := s.taskRepo.VerifyTaskListOwner(userID, taskListID); err != nil {
		return nil, fmt.Errorf("task list not found: %w", err)
	}
	return s.taskRepo.GetTasks(taskListID)
}

// SyncTasks picks the right strategy based on whether we've synced this list before.
func (s *service) SyncTasks(ctx context.Context, userID uint, taskListID string) error {
	if err := s.taskRepo.VerifyTaskListOwner(userID, taskListID); err != nil {
		return fmt.Errorf("task list not found: %w", err)
	}

	// Look up when we last synced this list
	list, err := s.taskRepo.GetTaskList(taskListID)
	if err != nil {
		return fmt.Errorf("failed to get task list: %w", err)
	}

	srv, err := s.getGoogleTaskService(ctx, userID)
	if err != nil {
		return err
	}

	if list.LastSync.IsZero() {
		return s.fullSync(srv, taskListID)
	}
	return s.incrementalSync(srv, taskListID, list.LastSync)
}

// fullSync fetches every active task from Google and replaces local state.
// Used on the very first sync of a list (when LastSync is zero).
func (s *service) fullSync(srv *tasks.Service, taskListID string) error {
	slog.Debug("performing full sync", "listID", taskListID)

	gTasks, err := srv.Tasks.List(taskListID).ShowCompleted(false).ShowHidden(true).Do()
	if err != nil {
		if s.isTokenError(err) {
			return errors.New("unauthorized: refresh token invalid")
		}
		return err
	}

	domainTasks, activeIDs := mapGoogleTasks(gTasks.Items, taskListID)
	sorted := topoSort(domainTasks)

	// Transactional write: upsert all tasks, mark missing ones as completed
	tx := s.taskRepo.BeginTx()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	txRepo := s.taskRepo.WithTx(tx)

	if err := txRepo.UpsertTasks(sorted); err != nil {
		tx.Rollback()
		return err
	}

	if err := txRepo.MarkTasksCompletedExcluding(taskListID, activeIDs); err != nil {
		tx.Rollback()
		return err
	}

	if err := txRepo.UpdateListLastSync(taskListID, time.Now()); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// incrementalSync fetches only tasks modified since lastSync.
// Handles updates, completions, and deletions in a single pass.
func (s *service) incrementalSync(srv *tasks.Service, taskListID string, lastSync time.Time) error {
	slog.Debug("performing incremental sync", "listID", taskListID, "since", lastSync)

	// Fetch changes since last sync.
	// ShowCompleted=true  → catches tasks completed since lastSync
	// ShowDeleted=true    → catches tasks deleted since lastSync (returned with Deleted=true)
	// ShowHidden=true     → catches hidden tasks (same as before)
	gTasks, err := srv.Tasks.List(taskListID).
		UpdatedMin(lastSync.Format(time.RFC3339)).
		ShowCompleted(true).
		ShowDeleted(true).
		ShowHidden(true).
		Do()
	if err != nil {
		if s.isTokenError(err) {
			return errors.New("unauthorized: refresh token invalid")
		}
		return err
	}

	// Nothing changed since last sync
	if len(gTasks.Items) == 0 {
		slog.Debug("incremental sync: no changes", "listID", taskListID)
		return s.taskRepo.UpdateListLastSync(taskListID, time.Now())
	}

	// Separate the changes into three buckets
	var toUpsert []domain.Task
	var toDelete []string

	for _, item := range gTasks.Items {
		// Google returns deleted tasks with Deleted=true
		if item.Deleted {
			toDelete = append(toDelete, item.Id)
			continue
		}

		task := mapSingleGoogleTask(item, taskListID)
		toUpsert = append(toUpsert, task)
	}

	// Sort upserts so parents come before children (prevents FK errors)
	sorted := topoSort(toUpsert)

	slog.Debug("incremental sync delta",
		"listID", taskListID,
		"upserts", len(sorted),
		"deletes", len(toDelete),
	)

	// Apply changes in a transaction
	tx := s.taskRepo.BeginTx()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	txRepo := s.taskRepo.WithTx(tx)

	if err := txRepo.DeleteTasks(toDelete); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete tasks: %w", err)
	}

	if err := txRepo.UpsertTasks(sorted); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to upsert tasks: %w", err)
	}

	if err := txRepo.UpdateListLastSync(taskListID, time.Now()); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// mapSingleGoogleTask converts a single Google Task item to a domain.Task.
func mapSingleGoogleTask(item *tasks.Task, taskListID string) domain.Task {
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

	return domain.Task{
		ID:         item.Id,
		Parent:     parent,
		TaskListID: taskListID,
		Title:      item.Title,
		Status:     item.Status,
		Notes:      item.Notes,
		Updated:    item.Updated,
		Due:        dueDate,
	}
}

// mapGoogleTasks converts a slice of Google Task items to domain tasks,
// filtering out completed tasks. Returns the domain tasks and their IDs.
// Used only by fullSync where we want active tasks only.
func mapGoogleTasks(items []*tasks.Task, taskListID string) ([]domain.Task, []string) {
	var domainTasks []domain.Task
	var activeIDs []string

	for _, item := range items {
		if item.Status != "needsAction" {
			continue
		}

		task := mapSingleGoogleTask(item, taskListID)
		domainTasks = append(domainTasks, task)
		activeIDs = append(activeIDs, item.Id)
	}

	return domainTasks, activeIDs
}

// topoSort orders tasks so that parents are inserted before children.
// Prevents FK constraint violations during upsert.
func topoSort(tasks []domain.Task) []domain.Task {
	if len(tasks) == 0 {
		return tasks
	}

	pendingMap := make(map[string]domain.Task, len(tasks))
	for _, t := range tasks {
		pendingMap[t.ID] = t
	}

	var sorted []domain.Task

	for len(pendingMap) > 0 {
		progress := false
		for id, t := range pendingMap {
			if t.Parent == nil {
				sorted = append(sorted, t)
				delete(pendingMap, id)
				progress = true
			} else if _, parentStillPending := pendingMap[*t.Parent]; !parentStillPending {
				sorted = append(sorted, t)
				delete(pendingMap, id)
				progress = true
			}
		}

		// Break cycles: orphaned children whose parents aren't in this batch
		if !progress {
			for id, t := range pendingMap {
				t.Parent = nil
				sorted = append(sorted, t)
				delete(pendingMap, id)
			}
		}
	}

	return sorted
}

func (s *service) UpdateTask(ctx context.Context, userID uint, taskID string, req domain.UpdateTaskRequest) error {
	task, err := s.taskRepo.GetTaskByID(taskID)
	if err != nil {
		return err
	}

	if err := s.taskRepo.VerifyTaskListOwner(userID, task.TaskListID); err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	// DETECT MOVE
	if req.TaskListID != nil && *req.TaskListID != task.TaskListID {
		return s.moveTask(ctx, userID, task, *req.TaskListID, req)
	}

	updates := make(map[string]interface{})

	if req.Quadrant != nil {
		updates["quadrant"] = *req.Quadrant
	}
	if req.DurationMins != nil {
		updates["duration_mins"] = *req.DurationMins
	}

	if req.Date != nil {
		if req.Date.Valid {
			updates["date"] = req.Date.Time
		} else {
			updates["date"] = nil
		}
	}

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

		updatedGTask, err := srv.Tasks.Patch(task.TaskListID, taskID, googleTask).Do()
		if err != nil {
			return fmt.Errorf("failed to sync to Google: %w", err)
		}

		updates["updated"] = updatedGTask.Updated
	}

	return s.taskRepo.UpdateTask(taskID, updates)
}

func (s *service) DeleteTask(ctx context.Context, userID uint, taskID string) error {
	task, err := s.taskRepo.GetTaskByID(taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	if err := s.taskRepo.VerifyTaskListOwner(userID, task.TaskListID); err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	srv, err := s.getGoogleTaskService(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get Google client: %w", err)
	}

	err = srv.Tasks.Delete(task.TaskListID, taskID).Do()
	if err != nil {
		fmt.Printf("Warning: Could not delete from Google Tasks: %v\n", err)
	}

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

// moveTask handles the "Delete Old + Create New" dance
func (s *service) moveTask(ctx context.Context, userID uint, currentTask domain.Task, newListID string, req domain.UpdateTaskRequest) error {
	if err := s.taskRepo.VerifyTaskListOwner(userID, newListID); err != nil {
		return fmt.Errorf("destination task list not found: %w", err)
	}

	srv, err := s.getGoogleTaskService(ctx, userID)
	if err != nil {
		return err
	}

	title := currentTask.Title
	if req.Title != nil {
		title = *req.Title
	}

	notes := currentTask.Notes
	if req.Notes != nil {
		notes = *req.Notes
	}

	status := currentTask.Status
	if req.Status != nil {
		status = *req.Status
	}

	googleTask := &tasks.Task{
		Title:  title,
		Notes:  notes,
		Status: status,
	}
	if req.Due != nil {
		googleTask.Due = req.Due.Format(time.RFC3339)
	}

	newGTask, err := srv.Tasks.Insert(newListID, googleTask).Do()
	if err != nil {
		return fmt.Errorf("failed to create task in new list: %w", err)
	}

	_ = srv.Tasks.Delete(currentTask.TaskListID, currentTask.ID).Do()

	newTask := domain.Task{
		ID:           newGTask.Id,
		TaskListID:   newListID,
		Title:        newGTask.Title,
		Status:       newGTask.Status,
		Updated:      newGTask.Updated,
		Notes:        newGTask.Notes,
		Quadrant:     currentTask.Quadrant,
		DurationMins: currentTask.DurationMins,
		IsRepeating:  currentTask.IsRepeating,
		Date:         currentTask.Date,
	}

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

	tx := s.taskRepo.BeginTx()

	if err := tx.Where("id = ?", currentTask.ID).Delete(&domain.Task{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&newTask).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
