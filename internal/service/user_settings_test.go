package service

import (
	"errors"
	"testing"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/nickquirk/life-dashboard-server/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSettingsService(repo *mocks.MockUserSettingsRepository) Service {
	return NewServiceWithRepos(nil, nil, nil, nil, nil, nil, nil, nil, repo)
}

// --- GetUserSettings ---

func TestService_GetUserSettings_Success(t *testing.T) {
	want := domain.Settings{
		Version: 1,
		Calendar: domain.CalendarSettings{
			StartHour: 7, EndHour: 21, DynamicRange: false,
		},
	}
	repo := &mocks.MockUserSettingsRepository{
		GetFunc: func(userID uint) (domain.Settings, error) {
			return want, nil
		},
	}
	svc := newSettingsService(repo)

	resp, err := svc.GetUserSettings(domain.GetUserSettingsRequest{UserID: 1})
	require.NoError(t, err)
	assert.Equal(t, want, resp.Settings)
}

func TestService_GetUserSettings_Error(t *testing.T) {
	repo := &mocks.MockUserSettingsRepository{
		GetFunc: func(userID uint) (domain.Settings, error) {
			return domain.Settings{}, errors.New("db error")
		},
	}
	svc := newSettingsService(repo)

	_, err := svc.GetUserSettings(domain.GetUserSettingsRequest{UserID: 1})
	assert.Error(t, err)
}

// --- UpdateUserSettings ---

func TestService_UpdateUserSettings_MergesPatch(t *testing.T) {
	repo := &mocks.MockUserSettingsRepository{}
	svc := newSettingsService(repo)

	resp, err := svc.UpdateUserSettings(domain.UpdateUserSettingsRequest{
		UserID: 1,
		Patch:  []byte(`{"calendar":{"startHour":7}}`),
	})
	require.NoError(t, err)
	assert.Equal(t, 7, resp.Calendar.StartHour)
	assert.Equal(t, domain.DefaultCalendarEndHour, resp.Calendar.EndHour)
}

func TestService_UpdateUserSettings_InvalidPatchRejected(t *testing.T) {
	cases := map[string]string{
		"end before start":  `{"calendar":{"startHour":18,"endHour":9}}`,
		"end equals start":  `{"calendar":{"startHour":9,"endHour":9}}`,
		"negative start":    `{"calendar":{"startHour":-1,"endHour":12}}`,
		"end past midnight": `{"calendar":{"startHour":9,"endHour":25}}`,
	}

	for name, patch := range cases {
		t.Run(name, func(t *testing.T) {
			persisted := false
			repo := &mocks.MockUserSettingsRepository{
				UpdateFunc: func(userID uint, mutate func(*domain.Settings) error) (domain.Settings, error) {
					cur := domain.DefaultSettings()
					err := mutate(&cur)
					if err == nil {
						persisted = true
					}
					return cur, err
				},
			}
			svc := newSettingsService(repo)

			_, err := svc.UpdateUserSettings(domain.UpdateUserSettingsRequest{UserID: 1, Patch: []byte(patch)})
			assert.ErrorIs(t, err, domain.ErrInvalidInput)
			assert.False(t, persisted, "mutate should not succeed for an invalid patch")
		})
	}
}

func TestService_UpdateUserSettings_RepoError(t *testing.T) {
	repo := &mocks.MockUserSettingsRepository{
		UpdateFunc: func(userID uint, mutate func(*domain.Settings) error) (domain.Settings, error) {
			return domain.Settings{}, errors.New("db error")
		},
	}
	svc := newSettingsService(repo)

	_, err := svc.UpdateUserSettings(domain.UpdateUserSettingsRequest{
		UserID: 1,
		Patch:  []byte(`{"calendar":{"startHour":8}}`),
	})
	assert.Error(t, err)
}
