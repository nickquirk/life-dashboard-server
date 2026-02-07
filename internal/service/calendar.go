package service

import (
	"context"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func (s *service) GetCalendarEvents(ctx context.Context, userID uint, start, end time.Time) ([]domain.GetCalendarEventsResponse, error) {
	// Initialise Calendar service
	svc, err := s.getGoogleCalendarService(ctx, userID)
	if err != nil {
		return []domain.GetCalendarEventsResponse{}, err
	}

	//filter to week that is viewed
	tMin := start.Format(time.RFC3339)
	tMax := end.Format(time.RFC3339)

	// hardcode to primary at the moment, this is the main cal for most people
	gEvents, err := svc.Events.List("primary").
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(tMin).
		TimeMax(tMax).
		Do()
	if err != nil {
		// could clean up tokens here?
		return nil, err
	}
	// map to domain
	var dto []domain.GetCalendarEventsResponse

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

		dto = {
			ID:       item.Id,
			Title:    item.Summary,
			Start:    startTime,
			End:      endTime,
			IsAllDay: isAllDay,
			ColorID:  item.ColorId,
		}
	}
	return dto, nil
}
