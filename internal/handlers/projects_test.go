package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// idRoute describes a handler that takes an {id} URL param. stub installs a
// service mock that records the IDs it was called with and returns err.
type idRoute struct {
	name     string
	handler  func(h *Handler) http.HandlerFunc
	method   string
	body     string // "" = no body expected
	stub     func(m *mocks.MockService, err error, gotID, gotUser *uint)
	wantJSON string
}

var idRoutes = []idRoute{
	{
		name: "updateProject", method: http.MethodPatch, body: `{"title":"Garden"}`,
		handler: func(h *Handler) http.HandlerFunc { return h.updateProject },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.UpdateProjectFunc = func(req domain.UpdateProjectRequest) (domain.Project, error) {
				*id, *user = req.ID, req.UserID
				return domain.Project{ID: req.ID, Title: *req.Title}, err
			}
		},
		wantJSON: `"title":"Garden"`,
	},
	{
		name: "deleteProject", method: http.MethodDelete,
		handler: func(h *Handler) http.HandlerFunc { return h.deleteProject },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.DeleteProjectFunc = func(req domain.DeleteProjectRequest) (domain.DeleteProjectResponse, error) {
				*id, *user = req.ID, req.UserID
				return domain.DeleteProjectResponse{ID: req.ID}, err
			}
		},
		wantJSON: `"id":5`,
	},
	{
		name: "getProjectActions", method: http.MethodGet,
		handler: func(h *Handler) http.HandlerFunc { return h.getProjectActions },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.GetActionsFunc = func(req domain.GetActionsRequest) (domain.GetActionsResponse, error) {
				*id, *user = req.ProjectID, req.UserID
				return domain.GetActionsResponse{Actions: []domain.Action{{ID: 1}}}, err
			}
		},
		wantJSON: `"actions":[`,
	},
	{
		name: "createAction", method: http.MethodPost, body: `{"title":"buy compost"}`,
		handler: func(h *Handler) http.HandlerFunc { return h.createAction },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.CreateActionFunc = func(req domain.CreateActionRequest) (domain.Action, error) {
				*id, *user = req.ProjectID, req.UserID
				return domain.Action{ID: 9, Title: req.Title, Stage: domain.StageBraindump}, err
			}
		},
		wantJSON: `"stage":"braindump"`,
	},
	{
		name: "reorderActions", method: http.MethodPut, body: `{"stage":"refine","orderedIds":[3,1]}`,
		handler: func(h *Handler) http.HandlerFunc { return h.reorderActions },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.ReorderActionsFunc = func(req domain.ReorderActionsRequest) (domain.ReorderActionsResponse, error) {
				*id, *user = req.ProjectID, req.UserID
				return domain.ReorderActionsResponse{ProjectID: req.ProjectID, Stage: req.Stage}, err
			}
		},
		wantJSON: `"stage":"refine"`,
	},
	{
		name: "updateAction", method: http.MethodPatch, body: `{"durationMins":null}`,
		handler: func(h *Handler) http.HandlerFunc { return h.updateAction },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.UpdateActionFunc = func(req domain.UpdateActionRequest) (domain.Action, error) {
				*id, *user = req.ID, req.UserID
				if !req.DurationMins.Set || req.DurationMins.Value != nil {
					return domain.Action{}, fmt.Errorf("explicit null not decoded")
				}
				return domain.Action{ID: req.ID}, err
			}
		},
		wantJSON: `"durationMins":null`,
	},
	{
		name: "changeActionStage", method: http.MethodPost, body: `{"stage":"ready"}`,
		handler: func(h *Handler) http.HandlerFunc { return h.changeActionStage },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.ChangeActionStageFunc = func(req domain.ChangeStageRequest) (domain.Action, error) {
				*id, *user = req.ID, req.UserID
				return domain.Action{ID: req.ID, Stage: req.Stage}, err
			}
		},
		wantJSON: `"stage":"ready"`,
	},
	{
		name: "scheduleAction", method: http.MethodPost, body: `{"date":"2026-10-17T09:00:00Z"}`,
		handler: func(h *Handler) http.HandlerFunc { return h.scheduleAction },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.ScheduleActionFunc = func(req domain.ScheduleActionRequest) (domain.Action, error) {
				*id, *user = req.ID, req.UserID
				return domain.Action{ID: req.ID, Stage: domain.StageScheduled, Date: req.Date}, err
			}
		},
		wantJSON: `"date":"2026-10-17T09:00:00Z"`,
	},
	{
		name: "moveAction", method: http.MethodPost, body: `{"date":"2026-10-18T14:00:00Z"}`,
		handler: func(h *Handler) http.HandlerFunc { return h.moveAction },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.MoveActionFunc = func(req domain.MoveActionRequest) (domain.Action, error) {
				*id, *user = req.ID, req.UserID
				return domain.Action{ID: req.ID, Stage: domain.StageScheduled, Date: req.Date}, err
			}
		},
		wantJSON: `"date":"2026-10-18T14:00:00Z"`,
	},
	{
		name: "unscheduleAction", method: http.MethodPost,
		handler: func(h *Handler) http.HandlerFunc { return h.unscheduleAction },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.UnscheduleActionFunc = func(req domain.UnscheduleActionRequest) (domain.Action, error) {
				*id, *user = req.ID, req.UserID
				return domain.Action{ID: req.ID, Stage: domain.StageReady}, err
			}
		},
		wantJSON: `"stage":"ready"`,
	},
	{
		name: "completeAction", method: http.MethodPost,
		handler: func(h *Handler) http.HandlerFunc { return h.completeAction },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.CompleteActionFunc = func(req domain.CompleteActionRequest) (domain.Action, error) {
				*id, *user = req.ID, req.UserID
				return domain.Action{ID: req.ID, Stage: domain.StageDone}, err
			}
		},
		wantJSON: `"stage":"done"`,
	},
	{
		name: "reopenAction", method: http.MethodPost,
		handler: func(h *Handler) http.HandlerFunc { return h.reopenAction },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.ReopenActionFunc = func(req domain.ReopenActionRequest) (domain.Action, error) {
				*id, *user = req.ID, req.UserID
				return domain.Action{ID: req.ID, Stage: domain.StageRefine}, err
			}
		},
		wantJSON: `"stage":"refine"`,
	},
	{
		name: "breakDownAction", method: http.MethodPost, body: `{"titles":["a","b"]}`,
		handler: func(h *Handler) http.HandlerFunc { return h.breakDownAction },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.BreakDownActionFunc = func(req domain.BreakDownActionRequest) (domain.GetActionsResponse, error) {
				*id, *user = req.ID, req.UserID
				return domain.GetActionsResponse{Actions: []domain.Action{{Title: req.Titles[0]}, {Title: req.Titles[1]}}}, err
			}
		},
		wantJSON: `"title":"b"`,
	},
	{
		name: "deleteAction", method: http.MethodDelete,
		handler: func(h *Handler) http.HandlerFunc { return h.deleteAction },
		stub: func(m *mocks.MockService, err error, id, user *uint) {
			m.DeleteActionFunc = func(req domain.DeleteActionRequest) (domain.DeleteActionResponse, error) {
				*id, *user = req.ID, req.UserID
				return domain.DeleteActionResponse{ID: req.ID}, err
			}
		},
		wantJSON: `"id":5`,
	},
}

func (rt idRoute) serve(svc *mocks.MockService, id string, body string, withAuth bool) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	r := httptest.NewRequest(rt.method, "/api/x/"+id, reader)
	r = withChi(r, "id", id)
	if withAuth {
		r = withUser(r, 1)
	}
	rr := httptest.NewRecorder()
	rt.handler(testHandler(svc))(rr, r)
	return rr
}

func TestProjectAndActionRoutes(t *testing.T) {
	for _, rt := range idRoutes {
		t.Run(rt.name+"/success", func(t *testing.T) {
			svc := &mocks.MockService{}
			var gotID, gotUser uint
			rt.stub(svc, nil, &gotID, &gotUser)
			rr := rt.serve(svc, "5", rt.body, true)
			assert.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
			assert.Contains(t, rr.Body.String(), rt.wantJSON)
			assert.Equal(t, uint(5), gotID)
			assert.Equal(t, uint(1), gotUser)
		})
		t.Run(rt.name+"/no user", func(t *testing.T) {
			rr := rt.serve(&mocks.MockService{}, "5", rt.body, false)
			assert.Equal(t, http.StatusUnauthorized, rr.Code)
		})
		t.Run(rt.name+"/bad id", func(t *testing.T) {
			rr := rt.serve(&mocks.MockService{}, "abc", rt.body, true)
			assert.Equal(t, http.StatusBadRequest, rr.Code)
		})
		if rt.body != "" {
			t.Run(rt.name+"/bad body", func(t *testing.T) {
				rr := rt.serve(&mocks.MockService{}, "5", "not json", true)
				assert.Equal(t, http.StatusBadRequest, rr.Code)
			})
		}
		t.Run(rt.name+"/invalid input", func(t *testing.T) {
			svc := &mocks.MockService{}
			var id, user uint
			rt.stub(svc, fmt.Errorf("%w: cannot do that in done", domain.ErrInvalidInput), &id, &user)
			rr := rt.serve(svc, "5", rt.body, true)
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), "cannot do that in done")
		})
		t.Run(rt.name+"/not found", func(t *testing.T) {
			svc := &mocks.MockService{}
			var id, user uint
			rt.stub(svc, gorm.ErrRecordNotFound, &id, &user)
			rr := rt.serve(svc, "5", rt.body, true)
			assert.Equal(t, http.StatusNotFound, rr.Code)
		})
		t.Run(rt.name+"/server error sanitised", func(t *testing.T) {
			svc := &mocks.MockService{}
			var id, user uint
			rt.stub(svc, errors.New("db exploded"), &id, &user)
			rr := rt.serve(svc, "5", rt.body, true)
			assert.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.NotContains(t, rr.Body.String(), "db exploded")
		})
	}
}

// --- createProject / getProjects ---

func TestCreateProject(t *testing.T) {
	var got domain.CreateProjectRequest
	svc := &mocks.MockService{
		CreateProjectFunc: func(req domain.CreateProjectRequest) (domain.Project, error) {
			got = req
			return domain.Project{ID: 2, Title: req.Title, Color: req.Color}, nil
		},
	}
	r := withUser(httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader([]byte(`{"title":"Garden","color":"green"}`))), 1)
	rr := httptest.NewRecorder()
	testHandler(svc).createProject(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"color":"green"`)
	assert.Equal(t, uint(1), got.UserID)
}

func TestCreateProject_Errors(t *testing.T) {
	cases := []struct {
		name string
		auth bool
		body string
		err  error
		want int
	}{
		{"no user", false, `{}`, nil, http.StatusUnauthorized},
		{"bad body", true, `nope`, nil, http.StatusBadRequest},
		{"invalid", true, `{}`, fmt.Errorf("%w: color must be a swatch", domain.ErrInvalidInput), http.StatusBadRequest},
		{"server", true, `{}`, errors.New("boom"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := &mocks.MockService{
				CreateProjectFunc: func(req domain.CreateProjectRequest) (domain.Project, error) { return domain.Project{}, c.err },
			}
			r := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader([]byte(c.body)))
			if c.auth {
				r = withUser(r, 1)
			}
			rr := httptest.NewRecorder()
			testHandler(svc).createProject(rr, r)
			assert.Equal(t, c.want, rr.Code)
		})
	}
}

func TestGetProjects_ArchivedParam(t *testing.T) {
	for query, want := range map[string]bool{"": false, "?archived=true": true, "?archived=no": false} {
		var got domain.GetProjectsRequest
		svc := &mocks.MockService{
			GetProjectsFunc: func(req domain.GetProjectsRequest) (domain.GetProjectsResponse, error) {
				got = req
				return domain.GetProjectsResponse{Projects: []domain.ProjectWithStats{}}, nil
			},
		}
		r := withUser(httptest.NewRequest(http.MethodGet, "/api/projects"+query, nil), 1)
		rr := httptest.NewRecorder()
		testHandler(svc).getProjects(rr, r)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), `"projects":[]`)
		assert.Equal(t, want, got.Archived, query)
	}
}

func TestGetProjects_NoUser(t *testing.T) {
	rr := httptest.NewRecorder()
	testHandler(&mocks.MockService{}).getProjects(rr, httptest.NewRequest(http.MethodGet, "/api/projects", nil))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// --- getActions ---

func TestGetActions_Range(t *testing.T) {
	var got domain.GetScheduledActionsRequest
	svc := &mocks.MockService{
		GetScheduledActionsFunc: func(req domain.GetScheduledActionsRequest) (domain.GetActionsWithProjectResponse, error) {
			got = req
			return domain.GetActionsWithProjectResponse{Actions: []domain.ActionWithProject{{
				Action:  domain.Action{ID: 1, Stage: domain.StageScheduled},
				Project: domain.ProjectRef{ID: 3, Title: "Garden", Color: "green"},
			}}}, nil
		},
	}
	r := withUser(httptest.NewRequest(http.MethodGet, "/api/actions?start=2026-10-12T00:00:00Z&end=2026-10-18T23:59:59Z", nil), 1)
	rr := httptest.NewRecorder()
	testHandler(svc).getActions(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"project":{"id":3,"title":"Garden","color":"green"}`)
	assert.True(t, got.Start.Equal(time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)))
	assert.True(t, got.End.Equal(time.Date(2026, 10, 18, 23, 59, 59, 0, time.UTC)))
	assert.Equal(t, uint(1), got.UserID)
}

func TestGetActions_Ready(t *testing.T) {
	called := false
	svc := &mocks.MockService{
		GetReadyActionsFunc: func(req domain.GetReadyActionsRequest) (domain.GetActionsWithProjectResponse, error) {
			called = true
			return domain.GetActionsWithProjectResponse{Actions: []domain.ActionWithProject{}}, nil
		},
	}
	r := withUser(httptest.NewRequest(http.MethodGet, "/api/actions?stage=ready", nil), 1)
	rr := httptest.NewRecorder()
	testHandler(svc).getActions(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called)
	assert.Contains(t, rr.Body.String(), `"actions":[]`)
}

func TestGetActions_BadQueries(t *testing.T) {
	for _, q := range []string{
		"",
		"?start=2026-10-12T00:00:00Z",
		"?end=2026-10-12T00:00:00Z",
		"?start=yesterday&end=2026-10-12T00:00:00Z",
		"?start=2026-10-12T00:00:00Z&end=tomorrow",
		"?stage=refine",
	} {
		r := withUser(httptest.NewRequest(http.MethodGet, "/api/actions"+q, nil), 1)
		rr := httptest.NewRecorder()
		testHandler(&mocks.MockService{}).getActions(rr, r)
		assert.Equal(t, http.StatusBadRequest, rr.Code, q)
	}
}

func TestGetActions_ServiceErrorSanitised(t *testing.T) {
	svc := &mocks.MockService{
		GetReadyActionsFunc: func(domain.GetReadyActionsRequest) (domain.GetActionsWithProjectResponse, error) {
			return domain.GetActionsWithProjectResponse{}, errors.New("db exploded")
		},
	}
	r := withUser(httptest.NewRequest(http.MethodGet, "/api/actions?stage=ready", nil), 1)
	rr := httptest.NewRecorder()
	testHandler(svc).getActions(rr, r)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.NotContains(t, rr.Body.String(), "db exploded")
}

func TestGetActions_NoUser(t *testing.T) {
	rr := httptest.NewRecorder()
	testHandler(&mocks.MockService{}).getActions(rr, httptest.NewRequest(http.MethodGet, "/api/actions?stage=ready", nil))
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
