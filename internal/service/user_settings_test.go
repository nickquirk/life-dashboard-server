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
	want := domain.Settings{Version: 1}
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

// There are no settings in the document today, so an empty patch is the only
// well-formed one. It still has to round-trip through the store.
func TestService_UpdateUserSettings_AppliesPatch(t *testing.T) {
	repo := &mocks.MockUserSettingsRepository{
		UpdateFunc: func(userID uint, mutate func(*domain.Settings) error) (domain.Settings, error) {
			cur := domain.DefaultSettings()
			return cur, mutate(&cur)
		},
	}
	svc := newSettingsService(repo)

	resp, err := svc.UpdateUserSettings(domain.UpdateUserSettingsRequest{
		UserID: 1,
		Patch:  []byte(`{}`),
	})
	require.NoError(t, err)
	assert.Equal(t, domain.DefaultSettings(), resp.Settings)
}

func TestService_UpdateUserSettings_InvalidPatchRejected(t *testing.T) {
	cases := map[string]string{
		"unknown field": `{"nope":1}`,
		"malformed":     `{`,
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
		Patch:  []byte(`{}`),
	})
	assert.Error(t, err)
}
