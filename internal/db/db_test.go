package db

import (
	"testing"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBackfillActionCompletedAt(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Project{}, &domain.Action{}, &domain.ActionGroup{}))

	day := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	kept := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	rows := []domain.Action{
		{Title: "done dated", Stage: domain.StageDone, Date: &day},
		{Title: "done undated", Stage: domain.StageDone},
		{Title: "done filled", Stage: domain.StageDone, Date: &day, CompletedAt: &kept},
		{Title: "ready", Stage: domain.StageReady},
	}
	for i := range rows {
		rows[i].UserID, rows[i].ProjectID = 1, 1
		require.NoError(t, db.Create(&rows[i]).Error)
	}

	require.NoError(t, BackfillActionCompletedAt(db))

	get := func(id uint) domain.Action {
		var a domain.Action
		require.NoError(t, db.First(&a, id).Error)
		return a
	}
	dated := get(rows[0].ID)
	require.NotNil(t, dated.CompletedAt)
	assert.True(t, dated.CompletedAt.Equal(day))
	assert.NotNil(t, get(rows[1].ID).CompletedAt, "undated falls back to updated_at")
	assert.True(t, get(rows[2].ID).CompletedAt.Equal(kept), "already filled is left alone")
	assert.Nil(t, get(rows[3].ID).CompletedAt)
}
