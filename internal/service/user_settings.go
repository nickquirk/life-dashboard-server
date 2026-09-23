package service

import (
	"github.com/nickquirk/life-dashboard-server/internal/domain"
)

func (s *service) GetUserSettings(req domain.GetUserSettingsRequest) (domain.UserSettingsResponse, error) {
	settings, err := s.settingsRepo.Get(req.UserID)
	if err != nil {
		return domain.UserSettingsResponse{}, err
	}
	return domain.UserSettingsResponse{Settings: settings}, nil
}

func (s *service) UpdateUserSettings(req domain.UpdateUserSettingsRequest) (domain.UserSettingsResponse, error) {
	settings, err := s.settingsRepo.Update(req.UserID, func(cur *domain.Settings) error {
		if err := cur.ApplyPatch(req.Patch); err != nil {
			return err
		}
		return cur.Validate()
	})
	if err != nil {
		return domain.UserSettingsResponse{}, err
	}
	return domain.UserSettingsResponse{Settings: settings}, nil
}
