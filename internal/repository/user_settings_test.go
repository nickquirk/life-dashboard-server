package repository

import (
	"errors"
	"testing"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newUserSettingsRepo(t *testing.T) *GormUserSettingsRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.UserSettings{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return &GormUserSettingsRepository{Db: db}
}

func TestUserSettingsRepo_GetMissingRowReturnsDefaults(t *testing.T) {
	repo := newUserSettingsRepo(t)

	got, err := repo.Get(testUserID)
	require.NoError(t, err)
	assert.Equal(t, domain.DefaultSettings(), got)
}

func TestUserSettingsRepo_UpdateCreatesThenUpdates(t *testing.T) {
	repo := newUserSettingsRepo(t)

	_, err := repo.Update(testUserID, func(s *domain.Settings) error {
		s.Calendar.StartHour = 7
		return nil
	})
	require.NoError(t, err)

	got, err := repo.Update(testUserID, func(s *domain.Settings) error {
		s.Calendar.DynamicRange = false
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 7, got.Calendar.StartHour)
	assert.False(t, got.Calendar.DynamicRange)

	var count int64
	require.NoError(t, repo.Db.Model(&domain.UserSettings{}).Where("user_id = ?", testUserID).Count(&count).Error)
	assert.Equal(t, int64(1), count, "update should not create a second row")
}

func TestUserSettingsRepo_UpdateMergesOverStoredDocument(t *testing.T) {
	repo := newUserSettingsRepo(t)

	_, err := repo.Update(testUserID, func(s *domain.Settings) error {
		s.Calendar.StartHour = 9
		s.Calendar.EndHour = 17
		return nil
	})
	require.NoError(t, err)

	_, err = repo.Update(testUserID, func(s *domain.Settings) error {
		s.Calendar.DynamicRange = false
		return nil
	})
	require.NoError(t, err)

	got, err := repo.Get(testUserID)
	require.NoError(t, err)
	assert.Equal(t, 9, got.Calendar.StartHour)
	assert.Equal(t, 17, got.Calendar.EndHour)
	assert.False(t, got.Calendar.DynamicRange)
}

func TestUserSettingsRepo_UpdatePersistsZeroValues(t *testing.T) {
	repo := newUserSettingsRepo(t)

	got, err := repo.Update(testUserID, func(s *domain.Settings) error {
		s.Calendar.StartHour = 0
		s.Calendar.DynamicRange = false
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 0, got.Calendar.StartHour)
	assert.False(t, got.Calendar.DynamicRange)

	fetched, err := repo.Get(testUserID)
	require.NoError(t, err)
	assert.Equal(t, 0, fetched.Calendar.StartHour)
	assert.False(t, fetched.Calendar.DynamicRange)
}

func TestUserSettingsRepo_UpdatePropagatesMutateError(t *testing.T) {
	repo := newUserSettingsRepo(t)

	wantErr := errors.New("mutate failed")
	_, err := repo.Update(testUserID, func(s *domain.Settings) error {
		return wantErr
	})
	assert.ErrorIs(t, err, wantErr)

	var count int64
	require.NoError(t, repo.Db.Model(&domain.UserSettings{}).Where("user_id = ?", testUserID).Count(&count).Error)
	assert.Equal(t, int64(0), count, "a failed mutate should leave no row behind")
}
