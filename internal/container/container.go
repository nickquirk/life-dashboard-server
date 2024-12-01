package container

import (
	"os"

	"github.com/nickquirk/life-dashboard-server/internal/helper"
	"github.com/nickquirk/life-dashboard-server/internal/service"
	"go.uber.org/dig"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func BuildContainer() *dig.Container {
	container := dig.New()
	container.Provide(NewApp)
	container.Provide(NewService)
	container.Provide(NewDb)

	return container
}

func NewApp() helper.App {
	return helper.NewApp("config.dev.yaml")
}

func NewService(db *gorm.DB) service.Service {
	return service.NewService(db)
}

func NewDb() *gorm.DB {
	dsn := os.Getenv("MARIA_DB")
	db, errConn := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if errConn != nil {
		panic(errConn)
	}

	return db
}
