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

// Version is the only field the document carries today, so these tests use it
// as the observable: what matters is that the stored document is loaded,
// handed to mutate, and written back to the same row.
func TestUserSettingsRepo_UpdateCreatesThenUpdates(t *testing.T) {
	repo := newUserSettingsRepo(t)

	_, err := repo.Update(testUserID, func(s *domain.Settings) error {
		s.Version = 7
		return nil
	})
	require.NoError(t, err)

	var seen int
	got, err := repo.Update(testUserID, func(s *domain.Settings) error {
		seen = s.Version // the second update must see the first one's write
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 7, seen)
	assert.Equal(t, 7, got.Version)

	var count int64
	require.NoError(t, repo.Db.Model(&domain.UserSettings{}).Where("user_id = ?", testUserID).Count(&count).Error)
	assert.Equal(t, int64(1), count, "update should not create a second row")
}

// Guards the GORM gotcha that silently dropped zero values from writes.
func TestUserSettingsRepo_UpdatePersistsZeroValues(t *testing.T) {
	repo := newUserSettingsRepo(t)

	got, err := repo.Update(testUserID, func(s *domain.Settings) error {
		s.Version = 0
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 0, got.Version)

	fetched, err := repo.Get(testUserID)
	require.NoError(t, err)
	assert.Equal(t, 0, fetched.Version)
}

// Guards the fix for a first-write gap-lock deadlock on MySQL: Update now
// upserts a placeholder row before taking the locking SELECT, so a user's
// very first write must still end up as exactly one row carrying the
// mutation, not a stray placeholder plus a second row.
func TestUserSettingsRepo_UpdateFirstWriteCreatesExactlyOneRow(t *testing.T) {
	repo := newUserSettingsRepo(t)

	got, err := repo.Update(testUserID, func(s *domain.Settings) error {
		s.Version = 42
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 42, got.Version)

	var count int64
	require.NoError(t, repo.Db.Model(&domain.UserSettings{}).Where("user_id = ?", testUserID).Count(&count).Error)
	assert.Equal(t, int64(1), count, "first write should leave exactly one row")
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
