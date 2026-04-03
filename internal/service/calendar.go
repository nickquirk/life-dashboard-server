package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func (s *service) GetCalendarEvents(ctx context.Context, req domain.GetCalendarEventsRequest) (domain.GetCalendarEventsResponse, error) {
	// Initialise Calendar service
	svc, err := s.getGoogleCalendarService(ctx, req.UserID)
	if err != nil {
		return domain.GetCalendarEventsResponse{}, err
	}

	//filter to week that is viewed
	tMin := req.Start.Format(time.RFC3339)
	tMax := req.End.Format(time.RFC3339)

	// hardcode to primary at the moment, this is the main cal for most people
	gEvents, err := svc.Events.List("primary").
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(tMin).
		TimeMax(tMax).
		Do()
	if err != nil {
		if s.isTokenError(err) {
			slog.Warn("unauthorized token during calendar event fetch", "userID", req.UserID)
			return domain.GetCalendarEventsResponse{}, errors.New("unauthorized: refresh token invalid")
		}
		return domain.GetCalendarEventsResponse{}, err
	}

	var events []domain.CalendarEvent

	for _, item := range gEvents.Items {
		var startTime, endTime time.Time
		var isAllDay bool

		if item.Start.DateTime != "" {
			startTime, _ = time.Parse(time.RFC3339, item.Start.DateTime)
			endTime, _ = time.Parse(time.RFC3339, item.End.DateTime)
		} else {
			startTime, _ = time.Parse("2006-01-02", item.Start.Date)
			endTime, _ = time.Parse("2006-01-02", item.End.Date)
			isAllDay = true
		}

		events = append(events, domain.CalendarEvent{
			ID:       item.Id,
			Title:    item.Summary,
			Start:    startTime,
			End:      endTime,
			IsAllDay: isAllDay,
			ColorID:  item.ColorId,
		})
	}

	dto := domain.GetCalendarEventsResponse{
		Events: events,
	}

	return dto, nil
}
