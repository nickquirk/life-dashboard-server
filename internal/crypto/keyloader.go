package crypto

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// LoadEncryptionKey loads a 32-byte AES-256 key.
// It checks the TOKEN_ENCRYPTION_KEY env var first (base64-encoded),
// then falls back to GCP Secret Manager in production.
func LoadEncryptionKey(gcpProjectID string) ([]byte, error) {
	if envKey := os.Getenv("TOKEN_ENCRYPTION_KEY"); envKey != "" {
		log.Println("Loading encryption key from environment variable")
		key, err := base64.StdEncoding.DecodeString(envKey)
		if err != nil {
			return nil, fmt.Errorf("failed to base64-decode TOKEN_ENCRYPTION_KEY: %w", err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("TOKEN_ENCRYPTION_KEY must decode to 32 bytes, got %d", len(key))
		}
		return key, nil
	}

	if os.Getenv("ENV") == "prod" {
		log.Println("Loading encryption key from GCP Secret Manager")
		return fetchFromSecretManager(gcpProjectID, "token-encryption-key")
	}

	return nil, fmt.Errorf("TOKEN_ENCRYPTION_KEY not set and not in prod environment")
}

func fetchFromSecretManager(projectID, secretID string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret manager client: %w", err)
	}
	defer client.Close()

	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretID)
	result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to access secret %s: %w", name, err)
	}

	// Try base64 decoding first
	key, err := base64.StdEncoding.DecodeString(string(result.Payload.Data))
	if err != nil {
		// Fall back to raw bytes if the secret was stored as raw 32 bytes
		if len(result.Payload.Data) == 32 {
			return result.Payload.Data, nil
		}
		return nil, fmt.Errorf("failed to decode secret payload: %w", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("secret must decode to 32 bytes, got %d", len(key))
	}
	return key, nil
}
