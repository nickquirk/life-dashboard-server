package service

import (
	"context"
	"fmt"
	"log"
	"sync"
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

	maxConcurrency := 5 // Sync 5 users at a time
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	// Loop and Launch
	for _, id := range userIDs {
		// Check if the parent context (request) is cancelled
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		sem <- struct{}{} // Acquire token (blocks if 5 are already running)

		go func(uid uint) {
			defer wg.Done()
			defer func() { <-sem }() // Release token

			// Create a specialized context for this user's work
			// This ensures one user timing out doesn't kill the whole job
			userCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()

			if err := s.syncSingleUser(userCtx, uid); err != nil {
				log.Printf("[Job] Error syncing user %d: %v", uid, err)
			}
		}(id)
	}

	// Wait for all to finish before returning
	wg.Wait()

	log.Println("[Job] Global sync complete")
	return nil
}

// syncSingleUser handles the logic for a single user (Lists + Tasks)
func (s *service) syncSingleUser(ctx context.Context, userID uint) error {
	// Sync Task Lists (Google -> DB)
	if err := s.SyncTaskLists(ctx, userID); err != nil {
		return fmt.Errorf("sync task lists: %w", err)
	}

	// Fetch the updated lists from DB
	lists, err := s.GetTaskLists(userID)
	if err != nil {
		return fmt.Errorf("get task lists: %w", err)
	}

	// Sync Tasks for each list
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
