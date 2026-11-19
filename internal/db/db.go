package db

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDb() (*gorm.DB, error) {
	dbConn := os.Getenv("DB_CONNECTION")
	if dbConn == "" {
		slog.Error("DB_CONNECTION environment variable is required")
		os.Exit(1)
	}

	var dialector gorm.Dialector

	switch dbConn {
	case "mysql":
		slog.Info("connecting to database", "driver", "mysql")
		user := os.Getenv("DB_USER")
		pass := os.Getenv("DB_PASS")
		host := os.Getenv("DB_HOST")
		dbName := os.Getenv("DB_NAME")
		var dsn string
		if os.Getenv("DB_SOCKET") != "" {
			// Unix Socket connection (Standard for Cloud Run)
			dsn = fmt.Sprintf("%s:%s@unix(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				user, pass, os.Getenv("DB_SOCKET"), dbName)
		} else {
			// TCP connection (Local or Private IP)
			dsn = fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				user, pass, host, dbName)
		}
		dialector = mysql.Open(dsn)
	default:
		slog.Error("unsupported DB_CONNECTION", "value", dbConn)
		os.Exit(1)
	}

	// Open the DB connection
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Retrieve the underlying sql.DB object to configure connection pooling
	sqlDB, err := db.DB()
	if err == nil {
		// Set maximum number of connections in the idle connection pool.
		sqlDB.SetMaxIdleConns(2)

		// Set maximum number of open connections to the database.
		// Crucial for Cloud Run to prevent connection exhaustion.
		sqlDB.SetMaxOpenConns(5)

		// Set maximum amount of time a connection may be reused.
		sqlDB.SetConnMaxLifetime(time.Hour)
	} else {
		slog.Warn("failed to set connection pool", "error", err)
	}

	return db, nil
}

// InitMigration migrates the schema. It stops at the first failure so a
// half-migrated schema fails the start instead of serving 500s.
func InitMigration(db *gorm.DB) error {
	steps := []struct {
		name   string
		models []any
	}{
		{"users", []any{&domain.User{}}},
		{"sessions", []any{&domain.Session{}}},
		{"task lists", []any{&domain.TaskList{}}},
		{"tasks", []any{&domain.Task{}}},
		{"zones", []any{&domain.Zone{}}},
		{"user settings", []any{&domain.UserSettings{}}},
		{"feedback", []any{&domain.Feedback{}}},
		{"scratchpads", []any{&domain.Scratchpad{}}},
		{"routines", []any{&domain.Routine{}}},
		{"routine instances", []any{&domain.RoutineInstance{}}},
		{"notes", []any{&domain.Note{}, &domain.NoteItem{}}},
		{"projects", []any{&domain.Project{}, &domain.Action{}, &domain.ActionGroup{}, &domain.WorkSession{}}},
	}
	for _, s := range steps {
		if err := db.AutoMigrate(s.models...); err != nil {
			return fmt.Errorf("migrate %s: %w", s.name, err)
		}
	}

	if err := BackfillActionCompletedAt(db); err != nil {
		return fmt.Errorf("backfill actions.completed_at: %w", err)
	}

	// Single-action scheduling was retired: work sessions are the only way
	// project work reaches the calendar. Runs once, while actions.date exists.
	if err := dropActionDate(db); err != nil {
		return err
	}

	// The single-session app_refresh_token column on users was replaced by the
	// sessions table. AutoMigrate doesn't drop columns, so do it explicitly.
	// Existing users will need to re-authenticate after this deploy.
	if db.Migrator().HasColumn(&domain.User{}, "app_refresh_token") {
		if err := db.Migrator().DropColumn(&domain.User{}, "app_refresh_token"); err != nil {
			return fmt.Errorf("drop users.app_refresh_token: %w", err)
		}
	}

	// One-time migration: target_total_mins -> goal_type/goal_target.
	// Safe to run repeatedly because the predicate excludes already-migrated rows.
	if db.Migrator().HasColumn(&domain.Routine{}, "target_total_mins") {
		if err := db.Exec(`
        UPDATE routines
        SET goal_type = 'time', goal_target = target_total_mins
        WHERE target_total_mins IS NOT NULL
          AND target_total_mins > 0
          AND goal_target IS NULL
    `).Error; err != nil {
			return fmt.Errorf("backfill routines goal columns: %w", err)
		}
		// Only drop the column once the backfill succeeded
		if err := db.Migrator().DropColumn(&domain.Routine{}, "target_total_mins"); err != nil {
			return fmt.Errorf("drop routines.target_total_mins: %w", err)
		}
	}
	return nil
}

// BackfillActionCompletedAt fills completed_at for actions done before the
// column existed. Approximate (date is the scheduled start, updated_at moves
// on any edit), but good enough for the Done column's "last 7 days". Safe to
// rerun because the predicate skips filled rows. It reads actions.date, so it
// only runs while that column exists: once more before dropActionDate drops
// it, and never on a fresh database.
func BackfillActionCompletedAt(db *gorm.DB) error {
	if !db.Migrator().HasColumn(&domain.Action{}, "date") {
		return nil
	}
	return db.Exec(`
		UPDATE actions
		SET completed_at = COALESCE(date, updated_at)
		WHERE stage = 'done' AND completed_at IS NULL
	`).Error
}

// dropActionDate moves every Scheduled action to the end of its project's
// Ready column, in calendar order, then drops actions.date and its index.
// Done actions keep their stage and completed_at. Guarded by the column, so
// it runs once; each step is safe to retry.
func dropActionDate(db *gorm.DB) error {
	m := db.Migrator()
	if !m.HasColumn(&domain.Action{}, "date") {
		return nil
	}

	// Raw table, because the struct no longer has Date; soft-deleted rows too
	err := db.Transaction(func(tx *gorm.DB) error {
		var rows []struct {
			ID        uint
			ProjectID uint
		}
		if err := tx.Table("actions").Select("id, project_id").
			Where("stage = ?", "scheduled").
			Order("project_id, date, id").Scan(&rows).Error; err != nil {
			return err
		}
		next := map[uint]int{}
		for _, r := range rows {
			pos, ok := next[r.ProjectID]
			if !ok {
				if err := tx.Table("actions").Select("COALESCE(MAX(position), -1) + 1").
					Where("project_id = ? AND stage = ?", r.ProjectID, "ready").
					Scan(&pos).Error; err != nil {
					return err
				}
			}
			if err := tx.Table("actions").Where("id = ?", r.ID).
				Updates(map[string]any{"stage": "ready", "position": pos}).Error; err != nil {
				return err
			}
			next[r.ProjectID] = pos + 1
		}
		slog.Info("moved scheduled actions to ready", "count", len(rows))
		return nil
	})
	if err != nil {
		return fmt.Errorf("move scheduled actions to ready: %w", err)
	}

	// Before the column, or MySQL quietly shrinks it to an index on user_id
	if m.HasIndex(&domain.Action{}, "idx_actions_user_date") {
		if err := m.DropIndex(&domain.Action{}, "idx_actions_user_date"); err != nil {
			return fmt.Errorf("drop idx_actions_user_date: %w", err)
		}
	}
	if err := m.DropColumn(&domain.Action{}, "date"); err != nil {
		return fmt.Errorf("drop actions.date: %w", err)
	}
	return nil
}
