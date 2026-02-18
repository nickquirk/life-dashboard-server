package db

import (
	"fmt"
	"log"
	"os"

	"github.com/nickquirk/life-dashboard-server/internal/domain"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDb() (*gorm.DB, error) {
	dbConn := os.Getenv("DB_CONNECTION")
	if dbConn == "" {
		dbConn = "sqlite" // Default to SQLite for local development
	}

	var dialector gorm.Dialector

	switch dbConn {
	case "mysql":
		log.Println("Connecting to MySQL (Production Mode)...")
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
	case "sqlite":
		log.Println("Connecting to SQLite (Local Mode)...")
		// SQLite creates a file named 'gorm.db' in the current directory.
		dialector = sqlite.Open("gorm.db")
	default:
		log.Fatalf("Unsupported DB_CONNECTION: %s. Use 'sqlite' or 'mysql'.", dbConn)
	}

	return gorm.Open(dialector, &gorm.Config{})
}

func InitMigration(db *gorm.DB) {
	db.AutoMigrate(domain.User{})
	db.AutoMigrate(domain.TaskList{})
	db.AutoMigrate(domain.Task{})
	db.AutoMigrate(domain.Zone{})
}
