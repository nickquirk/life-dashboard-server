package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scoped(id uint) domain.WorkSessionWithScope {
	return domain.WorkSessionWithScope{
		WorkSession: domain.WorkSession{ID: id, DurationMins: 60},
		Project:     domain.ProjectRef{ID: 3, Title: "Garden", Color: "green"},
	}
}

// workSessionRoutes run through the shared idRoute checks in TestProjectAndActionRoutes.
var workSessionRoutes = []idRoute{
	{
		name: "createWorkSession", method: http.MethodPost, body: `{"date":"2026-11-21T09:00:00Z","durationMins":60}`,
		handler: func(h *Handler) http.HandlerFunc { return h.createWorkSession },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.CreateWorkSessionFunc = func(req domain.CreateWorkSessionRequest) (domain.WorkSessionWithScope, error) {
				*id, *user = req.ProjectID, req.UserID
				return scoped(9), err
			}
		},
		wantJSON: `"durationMins":60`,
	},
	{
		name: "updateWorkSession", method: http.MethodPatch, body: `{"durationMins":90}`,
		handler: func(h *Handler) http.HandlerFunc { return h.updateWorkSession },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.UpdateWorkSessionFunc = func(req domain.UpdateWorkSessionRequest) (domain.WorkSessionWithScope, error) {
				*id, *user = req.ID, req.UserID
				ws := scoped(req.ID)
				ws.DurationMins = *req.DurationMins
				return ws, err
			}
		},
		wantJSON: `"durationMins":90`,
	},
	{
		name: "completeWorkSession", method: http.MethodPost,
		handler: func(h *Handler) http.HandlerFunc { return h.completeWorkSession },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.CompleteWorkSessionFunc = func(req domain.CompleteWorkSessionRequest) (domain.WorkSessionWithScope, error) {
				*id, *user = req.ID, req.UserID
				ws := scoped(req.ID)
				now := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
				ws.CompletedAt = &now
				return ws, err
			}
		},
		wantJSON: `"completedAt":"2026-11-21T10:00:00Z"`,
	},
	{
		name: "reopenWorkSession", method: http.MethodPost,
		handler: func(h *Handler) http.HandlerFunc { return h.reopenWorkSession },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.ReopenWorkSessionFunc = func(req domain.ReopenWorkSessionRequest) (domain.WorkSessionWithScope, error) {
				*id, *user = req.ID, req.UserID
				return scoped(req.ID), err
			}
		},
		wantJSON: `"completedAt":null`,
	},
	{
		name: "deleteWorkSession", method: http.MethodDelete,
		handler: func(h *Handler) http.HandlerFunc { return h.deleteWorkSession },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.DeleteWorkSessionFunc = func(req domain.DeleteWorkSessionRequest) (domain.DeleteWorkSessionResponse, error) {
				*id, *user = req.ID, req.UserID
				return domain.DeleteWorkSessionResponse{ID: req.ID}, err
			}
		},
		wantJSON: `"id":5`,
	},
}

func TestCreateWorkSession_DecodesBodyAndShape(t *testing.T) {
	var got domain.CreateWorkSessionRequest
	svc := &mocks.MockService{
		CreateWorkSessionFunc: func(req domain.CreateWorkSessionRequest) (domain.WorkSessionWithScope, error) {
			got = req
			return scoped(9), nil
		},
	}
	r := withUser(httptest.NewRequest(http.MethodPost, "/api/projects/3/work-sessions",
		bytes.NewReader([]byte(`{"groupId":4,"date":"2026-11-21T09:00:00Z","durationMins":45}`))), 1)
	r = withChi(r, "id", "3")
	rr := httptest.NewRecorder()
	testHandler(svc).createWorkSession(rr, r)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	assert.Equal(t, uint(3), got.ProjectID)
	require.NotNil(t, got.GroupID)
	assert.Equal(t, uint(4), *got.GroupID)
	assert.Equal(t, 45, got.DurationMins)
	assert.True(t, got.Date.Equal(time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)))
	body := rr.Body.String()
	for _, want := range []string{`"groupId":null`, `"durationMins":60`, `"completedAt":null`, `"project":{"id":3,"title":"Garden","color":"green"}`, `"group":null`} {
		assert.Contains(t, body, want)
	}
}

func TestGetWorkSessions_Range(t *testing.T) {
	var got domain.GetWorkSessionsRequest
	svc := &mocks.MockService{
		GetWorkSessionsFunc: func(req domain.GetWorkSessionsRequest) (domain.GetWorkSessionsResponse, error) {
			got = req
			ws := scoped(1)
			ws.Group = &domain.GroupRef{ID: 4, Title: "Beds"}
			return domain.GetWorkSessionsResponse{WorkSessions: []domain.WorkSessionWithScope{ws}}, nil
		},
	}
	r := withUser(httptest.NewRequest(http.MethodGet, "/api/work-sessions?start=2026-11-16T00:00:00Z&end=2026-11-22T23:59:59Z", nil), 1)
	rr := httptest.NewRecorder()
	testHandler(svc).getWorkSessions(rr, r)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"workSessions":[`)
	assert.Contains(t, rr.Body.String(), `"group":{"id":4,"title":"Beds"}`)
	assert.True(t, got.Start.Equal(time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)))
	assert.True(t, got.End.Equal(time.Date(2026, 11, 22, 23, 59, 59, 0, time.UTC)))
	assert.Equal(t, uint(1), got.UserID)
}

func TestGetWorkSessions_BadQueries(t *testing.T) {
	for _, q := range []string{
		"",
		"?start=2026-11-16T00:00:00Z",
		"?end=2026-11-16T00:00:00Z",
		"?start=yesterday&end=2026-11-16T00:00:00Z",
		"?start=2026-11-16T00:00:00Z&end=tomorrow",
	} {
		r := withUser(httptest.NewRequest(http.MethodGet, "/api/work-sessions"+q, nil), 1)
		rr := httptest.NewRecorder()
		testHandler(&mocks.MockService{}).getWorkSessions(rr, r)
		assert.Equal(t, http.StatusBadRequest, rr.Code, q)
	}
}

func TestGetWorkSessions_NoUser(t *testing.T) {
	rr := httptest.NewRecorder()
	testHandler(&mocks.MockService{}).getWorkSessions(rr, httptest.NewRequest(http.MethodGet, "/api/work-sessions", nil))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
