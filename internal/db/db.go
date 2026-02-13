package db

import (
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
		// NOTE: Replace this DSN with your actual production credentials,
		// typically loaded from environment variables (DB_USER, DB_PASS, etc.).
		// Format: user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
		dsn := "user:password@tcp(127.0.0.1:3306)/gorm_prod_db?charset=utf8mb4&parseTime=True&loc=Local"
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
