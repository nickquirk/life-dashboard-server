package service

import (
	"context"
	"encoding/json"
	"fmt"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
)

// TriggerGlobalSync finds all users and enqueues a Cloud Task for each.
func (s *service) TriggerGlobalSync(ctx context.Context, projectID, location, queue, workerURL, serviceAccountEmail string) error {
	userIDs, err := s.userRepo.GetUsersWithRefreshTokens()
	if err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("cloud tasks client: %w", err)
	}
	defer client.Close()

	queuePath := fmt.Sprintf("projects/%s/locations/%s/queues/%s", projectID, location, queue)

	for _, uid := range userIDs {
		// Create the payload for the worker
		payload, _ := json.Marshal(map[string]uint{"user_id": uid})

		req := &taskspb.CreateTaskRequest{
			Parent: queuePath,
			Task: &taskspb.Task{
				MessageType: &taskspb.Task_HttpRequest{
					HttpRequest: &taskspb.HttpRequest{
						HttpMethod: taskspb.HttpMethod_POST,
						Url:        workerURL,
						Body:       payload,
						Headers:    map[string]string{"Content-Type": "application/json"},
						// This ensures Cloud Tasks authenticates with your worker!
						AuthorizationHeader: &taskspb.HttpRequest_OidcToken{
							OidcToken: &taskspb.OidcToken{
								ServiceAccountEmail: serviceAccountEmail,
							},
						},
					},
				},
			},
		}

		if _, err := client.CreateTask(ctx, req); err != nil {
			// Log the error, but continue enqueueing other users
			fmt.Printf("failed to enqueue task for user %d: %v\n", uid, err)
		}
	}

	return nil
}

// SyncSingleUser remains mostly the same, but expose it so the handler can call it directly.
func (s *service) SyncSingleUser(ctx context.Context, userID uint) error {
	// Your existing syncSingleUser logic goes here...
	return nil
}
