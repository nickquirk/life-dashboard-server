package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// --- createRoutine ---

func TestCreateRoutine_Success(t *testing.T) {
	svc := &mocks.MockService{
		CreateRoutineFunc: func(req domain.CreateRoutineRequest) (domain.CreateRoutineResponse, error) {
			return domain.CreateRoutineResponse{ID: 7, Title: req.Title, DurationMins: req.DurationMins}, nil
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.CreateRoutineRequest{Title: "Read", DurationMins: 15})
	r := httptest.NewRequest(http.MethodPost, "/api/routines", bytes.NewReader(body))
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.createRoutine(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"id":7`)
}

func TestCreateRoutine_NoUser(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPost, "/api/routines", nil)
	rr := httptest.NewRecorder()

	h.createRoutine(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestCreateRoutine_BadBody(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPost, "/api/routines", bytes.NewReader([]byte("not json")))
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.createRoutine(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateRoutine_ValidationError_Returns400AndExposesMessage(t *testing.T) {
	svc := &mocks.MockService{
		CreateRoutineFunc: func(req domain.CreateRoutineRequest) (domain.CreateRoutineResponse, error) {
			return domain.CreateRoutineResponse{}, fmt.Errorf("%w: title is required", domain.ErrInvalidInput)
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.CreateRoutineRequest{Title: "", DurationMins: 15})
	r := httptest.NewRequest(http.MethodPost, "/api/routines", bytes.NewReader(body))
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.createRoutine(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "title is required")
}

func TestCreateRoutine_DecodesGoalShape(t *testing.T) {
	// Verify a real `"goal": {"type":..., "target":...}` JSON payload reaches
	// the service as a populated *domain.Goal.
	var captured domain.CreateRoutineRequest
	svc := &mocks.MockService{
		CreateRoutineFunc: func(req domain.CreateRoutineRequest) (domain.CreateRoutineResponse, error) {
			captured = req
			return domain.CreateRoutineResponse{ID: 9, Title: req.Title, DurationMins: req.DurationMins}, nil
		},
	}
	h := testHandler(svc)
	body := []byte(`{"title":"Read","durationMins":15,"goal":{"type":"time","target":30}}`)
	r := httptest.NewRequest(http.MethodPost, "/api/routines", bytes.NewReader(body))
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.createRoutine(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	if assert.NotNil(t, captured.Goal) {
		assert.Equal(t, domain.GoalTypeTime, captured.Goal.Type)
		assert.Equal(t, 30, captured.Goal.Target)
	}
}

func TestCreateRoutine_ServiceError_Returns500(t *testing.T) {
	svc := &mocks.MockService{
		CreateRoutineFunc: func(req domain.CreateRoutineRequest) (domain.CreateRoutineResponse, error) {
			return domain.CreateRoutineResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	body, _ := json.Marshal(domain.CreateRoutineRequest{Title: "Read", DurationMins: 15})
	r := httptest.NewRequest(http.MethodPost, "/api/routines", bytes.NewReader(body))
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.createRoutine(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NotContains(t, rr.Body.String(), "db error")
}

// --- updateRoutine ---

func TestUpdateRoutine_Success(t *testing.T) {
	svc := &mocks.MockService{
		UpdateRoutineFunc: func(req domain.UpdateRoutineRequest) (domain.UpdateRoutineResponse, error) {
			return domain.UpdateRoutineResponse{ID: req.ID}, nil
		},
	}
	h := testHandler(svc)
	title := "New"
	body, _ := json.Marshal(domain.UpdateRoutineRequest{Title: &title})
	r := httptest.NewRequest(http.MethodPatch, "/api/routines/5", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.updateRoutine(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"id":5`)
}

func TestUpdateRoutine_InvalidID(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPatch, "/api/routines/abc", nil)
	r = withUserAndChi(r, 1, "id", "abc")
	rr := httptest.NewRecorder()

	h.updateRoutine(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateRoutine_BadBody(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodPatch, "/api/routines/5", bytes.NewReader([]byte("bad")))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.updateRoutine(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateRoutine_ValidationError_Returns400(t *testing.T) {
	svc := &mocks.MockService{
		UpdateRoutineFunc: func(req domain.UpdateRoutineRequest) (domain.UpdateRoutineResponse, error) {
			return domain.UpdateRoutineResponse{}, fmt.Errorf("%w: durationMins must be between 1 and 1440", domain.ErrInvalidInput)
		},
	}
	h := testHandler(svc)
	title := "x"
	body, _ := json.Marshal(domain.UpdateRoutineRequest{Title: &title})
	r := httptest.NewRequest(http.MethodPatch, "/api/routines/5", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.updateRoutine(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "durationMins must be between")
}

func TestUpdateRoutine_GoalSet_PassedToService(t *testing.T) {
	// `"goal": {"type":..., "target":...}` reaches the service as
	// GoalUpdate{Set: true, Value: <populated>}.
	var captured domain.UpdateRoutineRequest
	svc := &mocks.MockService{
		UpdateRoutineFunc: func(req domain.UpdateRoutineRequest) (domain.UpdateRoutineResponse, error) {
			captured = req
			return domain.UpdateRoutineResponse{ID: req.ID}, nil
		},
	}
	h := testHandler(svc)
	body := []byte(`{"goal":{"type":"count","target":5}}`)
	r := httptest.NewRequest(http.MethodPatch, "/api/routines/5", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.updateRoutine(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, captured.Goal.Set)
	if assert.NotNil(t, captured.Goal.Value) {
		assert.Equal(t, domain.GoalTypeCount, captured.Goal.Value.Type)
		assert.Equal(t, 5, captured.Goal.Value.Target)
	}
}

func TestUpdateRoutine_GoalNull_PassedToServiceAsClear(t *testing.T) {
	// `"goal": null` must reach the service as GoalUpdate{Set: true, Value: nil}
	// — the clear-goal sentinel that drives the demote-to-regular-routine path.
	var captured domain.UpdateRoutineRequest
	svc := &mocks.MockService{
		UpdateRoutineFunc: func(req domain.UpdateRoutineRequest) (domain.UpdateRoutineResponse, error) {
			captured = req
			return domain.UpdateRoutineResponse{ID: req.ID}, nil
		},
	}
	h := testHandler(svc)
	body := []byte(`{"goal":null}`)
	r := httptest.NewRequest(http.MethodPatch, "/api/routines/5", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.updateRoutine(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, captured.Goal.Set, "goal:null must set the Set flag")
	assert.Nil(t, captured.Goal.Value, "goal:null must produce nil Value (the clear sentinel)")
}

func TestUpdateRoutine_GoalAbsent_NotPassed(t *testing.T) {
	// Body without "goal" must reach the service as GoalUpdate{Set: false} —
	// the "no change" signal. The service must NOT touch goal columns.
	var captured domain.UpdateRoutineRequest
	svc := &mocks.MockService{
		UpdateRoutineFunc: func(req domain.UpdateRoutineRequest) (domain.UpdateRoutineResponse, error) {
			captured = req
			return domain.UpdateRoutineResponse{ID: req.ID}, nil
		},
	}
	h := testHandler(svc)
	body := []byte(`{"title":"New"}`)
	r := httptest.NewRequest(http.MethodPatch, "/api/routines/5", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.updateRoutine(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.False(t, captured.Goal.Set, "absent goal must leave Set==false")
	assert.Nil(t, captured.Goal.Value)
}

func TestUpdateRoutine_NotFound(t *testing.T) {
	svc := &mocks.MockService{
		UpdateRoutineFunc: func(req domain.UpdateRoutineRequest) (domain.UpdateRoutineResponse, error) {
			return domain.UpdateRoutineResponse{}, gorm.ErrRecordNotFound
		},
	}
	h := testHandler(svc)
	title := "x"
	body, _ := json.Marshal(domain.UpdateRoutineRequest{Title: &title})
	r := httptest.NewRequest(http.MethodPatch, "/api/routines/999", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "999")
	rr := httptest.NewRecorder()

	h.updateRoutine(rr, r)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestUpdateRoutine_ServiceError_Returns500(t *testing.T) {
	svc := &mocks.MockService{
		UpdateRoutineFunc: func(req domain.UpdateRoutineRequest) (domain.UpdateRoutineResponse, error) {
			return domain.UpdateRoutineResponse{}, errors.New("db error")
		},
	}
	h := testHandler(svc)
	title := "x"
	body, _ := json.Marshal(domain.UpdateRoutineRequest{Title: &title})
	r := httptest.NewRequest(http.MethodPatch, "/api/routines/5", bytes.NewReader(body))
	r = withUserAndChi(r, 1, "id", "5")
	rr := httptest.NewRecorder()

	h.updateRoutine(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// --- createRoutineInstance ---

func TestCreateRoutineInstance_PassesDurationMins(t *testing.T) {
	var captured domain.CreateRoutineInstanceRequest
	svc := &mocks.MockService{
		CreateRoutineInstanceFunc: func(req domain.CreateRoutineInstanceRequest) (domain.CreateRoutineInstanceResponse, error) {
			captured = req
			return domain.CreateRoutineInstanceResponse{ID: 1, RoutineID: req.RoutineID, DurationMins: req.DurationMins, Date: req.Date, Status: "needsAction"}, nil
		},
	}
	h := testHandler(svc)
	body := []byte(`{"routineId":5,"date":"2026-01-01T00:00:00Z","durationMins":45}`)
	r := httptest.NewRequest(http.MethodPost, "/api/routine-instances", bytes.NewReader(body))
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.createRoutineInstance(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	if assert.NotNil(t, captured.DurationMins) {
		assert.Equal(t, 45, *captured.DurationMins)
	}
	assert.Equal(t, uint(5), captured.RoutineID)
}
