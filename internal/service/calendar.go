package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"google.golang.org/api/calendar/v3"
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

	events := make([]domain.CalendarEvent, 0)

	for _, item := range gEvents.Items {
		startTime, endTime, isAllDay, err := parseEventTimes(item)
		if err != nil {
			slog.Warn("skipping calendar event with unparseable time",
				"userID", req.UserID, "eventID", item.Id, "error", err)
			continue
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

func parseEventTimes(item *calendar.Event) (start, end time.Time, isAllDay bool, err error) {
	if item.Start.DateTime != "" {
		start, err = time.Parse(time.RFC3339, item.Start.DateTime)
		if err != nil {
			return time.Time{}, time.Time{}, false, fmt.Errorf("start datetime %q: %w", item.Start.DateTime, err)
		}
		end, err = time.Parse(time.RFC3339, item.End.DateTime)
		if err != nil {
			return time.Time{}, time.Time{}, false, fmt.Errorf("end datetime %q: %w", item.End.DateTime, err)
		}
		return start, end, false, nil
	}
	start, err = time.Parse("2006-01-02", item.Start.Date)
	if err != nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("start date %q: %w", item.Start.Date, err)
	}
	end, err = time.Parse("2006-01-02", item.End.Date)
	if err != nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("end date %q: %w", item.End.Date, err)
	}
	return start, end, true, nil
}
