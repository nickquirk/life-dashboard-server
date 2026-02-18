package service

import (
	"context"
	"fmt"
	"log"
	"time"
)

// SyncAllUsers iterates through all users with a refresh token and syncs their data.
func (s *service) SyncAllUsers(ctx context.Context) error {
	log.Println("[Job] Starting global sync...")

	userIDs, err := s.userRepo.GetUsersWithRefreshTokens()
	if err != nil {
		return fmt.Errorf("failed to fetch users for sync: %w", err)
	}

	log.Printf("[Job] Found %d users to sync", len(userIDs))

	var success, errors int

	// 2. Iterate and sync each user
	for _, userID := range userIDs {
		// Create a separate timeout for each user so one slow sync doesn't kill the whole job immediately
		userCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)

		err := s.syncSingleUser(userCtx, userID)
		cancel() // Clean up context immediately

		if err != nil {
			log.Printf("[Job] Error syncing user %d: %v", userID, err)
			errors++
			continue
		}
		success++
	}

	log.Printf("[Job] Sync complete. Success: %d, Errors: %d", success, errors)
	return nil
}

// syncSingleUser handles the logic for a single user (Lists + Tasks)
func (s *service) syncSingleUser(ctx context.Context, userID uint) error {
	// A. Sync Task Lists (Google -> DB)
	if err := s.SyncTaskLists(ctx, userID); err != nil {
		return fmt.Errorf("sync task lists: %w", err)
	}

	// B. Fetch the updated lists from DB
	lists, err := s.GetTaskLists(userID)
	if err != nil {
		return fmt.Errorf("get task lists: %w", err)
	}

	// C. Sync Tasks for each list
	for _, list := range lists {
		// Check context before starting next list (in case of timeout/shutdown)
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err := s.SyncTasks(ctx, userID, list.ID); err != nil {
			// Log but continue to next list (don't fail the whole user for one list)
			log.Printf("[Job] Warning: Failed to sync list %s for user %d: %v", list.ID, userID, err)
		}
	}

	return nil
}
