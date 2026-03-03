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

func InitMigration(db *gorm.DB) {
	db.AutoMigrate(domain.User{})
	db.AutoMigrate(domain.TaskList{})
	db.AutoMigrate(domain.Task{})
	db.AutoMigrate(domain.Zone{})
	db.AutoMigrate(&domain.Feedback{})
}
