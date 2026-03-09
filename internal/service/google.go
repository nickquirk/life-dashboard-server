package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nickquirk/life-dashboard-server/internal/config"
	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

// Helper to get Google Client
func (s *service) getAuthenticatedClient(ctx context.Context, userID uint) (*http.Client, error) {
	//  Get User for tokens
	userResp, err := s.userRepo.Get(domain.GetUserRequest{ID: userID})
	if err != nil {
		return nil, err
	}

	// Build Token
	tok := &oauth2.Token{
		AccessToken:  userResp.AccessToken,
		RefreshToken: userResp.RefreshToken,
		Expiry:       userResp.TokenExpiry,
		TokenType:    "Bearer",
	}

	// Create generic HTTP Client
	conf := config.GetGoogleConfig()
	return conf.Client(ctx, tok), nil
}

func (s *service) getGoogleTaskService(ctx context.Context, userID uint) (*tasks.Service, error) {
	client, err := s.getAuthenticatedClient(ctx, userID)
	if err != nil {
		return &tasks.Service{}, fmt.Errorf("failed to create tasks service: %w", err)
	}
	return tasks.NewService(ctx, option.WithHTTPClient(client))
}

func (s *service) getGoogleCalendarService(ctx context.Context, userID uint) (*calendar.Service, error) {
	client, err := s.getAuthenticatedClient(ctx, userID)
	if err != nil {
		return &calendar.Service{}, fmt.Errorf("failed to create calendar service: %w", err)
	}
	return calendar.NewService(ctx, option.WithHTTPClient(client))
}

func (s *service) getZonesService(ctx context.Context, userID uint) (*calendar.Service, error) {
	client, err := s.getAuthenticatedClient(ctx, userID)
	if err != nil {
		return &calendar.Service{}, fmt.Errorf("failed to create zones service: %w", err)
	}
	return calendar.NewService(ctx, option.WithHTTPClient(client))
}
