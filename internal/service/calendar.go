package service

import (
	"context"
	"fmt"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func (s *service) GetCalendarEvents(ctx context.Context, userID uint, start, end time.Time) ([]domain.CalendarEvent, error) {
	client, err := s.getGoogleClient(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create google client: %w", err)
	}

	// Initialise Calendar service
	svc, err := s.getGoogleCalendarService(ctx, userID)
	if err != nil {
		return []domain.CalendarEvent{}, err
	}

}
