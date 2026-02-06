package service

import (
	"context"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func (s *service) GetCalendarEvents(ctx context.Context, userID uint, start, end time.Time) ([]domain.CalendarEvent, error) {
	// Initialise Calendar service
	svc, err := s.getGoogleCalendarService(ctx, userID)
	if err != nil {
		return []domain.CalendarEvent{}, err
	}

}
