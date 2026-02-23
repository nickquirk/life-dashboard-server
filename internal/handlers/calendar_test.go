package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
)

func TestGetCalendarEvents_Success(t *testing.T) {
	svc := &mocks.MockService{
		GetCalendarEventsFunc: func(ctx context.Context, req domain.GetCalendarEventsRequest) (domain.GetCalendarEventsResponse, error) {
			return domain.GetCalendarEventsResponse{
				Events: []domain.CalendarEvent{
					{ID: "e1", Title: "Meeting", Start: time.Now(), End: time.Now().Add(time.Hour)},
				},
			}, nil
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/api/calendar/events?start=2024-01-01T00:00:00Z&end=2024-01-07T00:00:00Z", nil)
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.getCalendarEvents(rr, r)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Meeting")
}

func TestGetCalendarEvents_MissingStart(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodGet, "/api/calendar/events?end=2024-01-07T00:00:00Z", nil)
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.getCalendarEvents(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "start and end")
}

func TestGetCalendarEvents_InvalidFormat(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodGet, "/api/calendar/events?start=bad-date&end=2024-01-07T00:00:00Z", nil)
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.getCalendarEvents(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "RFC3339")
}

func TestGetCalendarEvents_InvalidEndFormat(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodGet, "/api/calendar/events?start=2024-01-01T00:00:00Z&end=bad-date", nil)
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.getCalendarEvents(rr, r)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "end param")
}

func TestGetCalendarEvents_NoUser(t *testing.T) {
	h := testHandler(&mocks.MockService{})
	r := httptest.NewRequest(http.MethodGet, "/api/calendar/events?start=2024-01-01T00:00:00Z&end=2024-01-07T00:00:00Z", nil)
	rr := httptest.NewRecorder()

	h.getCalendarEvents(rr, r)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestGetCalendarEvents_ServiceError(t *testing.T) {
	svc := &mocks.MockService{
		GetCalendarEventsFunc: func(ctx context.Context, req domain.GetCalendarEventsRequest) (domain.GetCalendarEventsResponse, error) {
			return domain.GetCalendarEventsResponse{}, errors.New("api error")
		},
	}
	h := testHandler(svc)
	r := httptest.NewRequest(http.MethodGet, "/api/calendar/events?start=2024-01-01T00:00:00Z&end=2024-01-07T00:00:00Z", nil)
	r = withUser(r, 1)
	rr := httptest.NewRecorder()

	h.getCalendarEvents(rr, r)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
