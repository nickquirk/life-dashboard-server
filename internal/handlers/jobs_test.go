package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
)

func TestHandleTriggerSync_Success(t *testing.T) {
	svc := &mocks.MockService{
		TriggerGlobalSyncFunc: func(ctx context.Context, projectID, location, queue, workerURL, serviceAccountEmail string) error {
			return nil
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodPost, "/api/jobs/sync/trigger", nil)
	rr := httptest.NewRecorder()

	h.handleTriggerSync(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "tasks_enqueued")
}

func TestHandleTriggerSync_Error(t *testing.T) {
	svc := &mocks.MockService{
		TriggerGlobalSyncFunc: func(ctx context.Context, projectID, location, queue, workerURL, serviceAccountEmail string) error {
			return errors.New("cloud tasks error")
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodPost, "/api/jobs/sync/trigger", nil)
	rr := httptest.NewRecorder()

	h.handleTriggerSync(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestHandleSyncWorker_Success(t *testing.T) {
	svc := &mocks.MockService{
		SyncSingleUserFunc: func(ctx context.Context, userID uint) error {
			assert.Equal(t, uint(42), userID)
			return nil
		},
	}
	h := testHandler(svc)
	payload, _ := json.Marshal(map[string]uint{"user_id": 42})
	r := httptest.NewRequest(http.MethodPost, "/api/jobs/sync/worker", bytes.NewReader(payload))
	rr := httptest.NewRecorder()

	h.handleSyncWorker(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleSyncWorker_InvalidPayload(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPost, "/api/jobs/sync/worker", bytes.NewReader([]byte("not json")))
	rr := httptest.NewRecorder()

	h.handleSyncWorker(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleSyncWorker_SyncError(t *testing.T) {
	svc := &mocks.MockService{
		SyncSingleUserFunc: func(ctx context.Context, userID uint) error {
			return errors.New("sync failed")
		},
	}
	h := testHandler(svc)
	payload, _ := json.Marshal(map[string]uint{"user_id": 1})
	r := httptest.NewRequest(http.MethodPost, "/api/jobs/sync/worker", bytes.NewReader(payload))
	rr := httptest.NewRecorder()

	h.handleSyncWorker(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
