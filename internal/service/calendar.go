package service

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	"github.com/nickquirk/life-dashboard-server/internal/domain"
// 	"google.golang.org/api/calendar/v3"
// 	"google.golang.org/api/option"
// )

// func (s *service) GetCalendarEvents(ctx context.Context, userID uint, start, end time.Time) ([]domain.CalendarEvent, error) {
// 	client, err := s.getGoogleClient(ctx, userID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create google client: %w", err)
// 	}

// 	// Initialise Calendar service
// 	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client ))
// 	return svc, nil
// }
