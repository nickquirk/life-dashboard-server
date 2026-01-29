package container

import (
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/db"
	"github.com/nickquirk/life-dashboard-server/internal/handlers"
	"github.com/nickquirk/life-dashboard-server/internal/helper"
	"github.com/nickquirk/life-dashboard-server/internal/poller"
	"github.com/nickquirk/life-dashboard-server/internal/service"
	"go.uber.org/dig"
	"gorm.io/gorm"
)

func BuildContainer() *dig.Container {
	container := dig.New()

	// Provide the dependencies
	container.Provide(NewApp)
	container.Provide(NewChiRouter)
	container.Provide(NewHandler)
	container.Provide(NewService)
	container.Provide(NewConnection)
	container.Provide(NewPoller)

	// Provide the Main Application Wrapper (defined in application.go)
	container.Provide(NewApplication)

	return container
}

func NewApp() helper.App {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.dev.yaml"
	}
	return helper.NewApp(configPath)
}

func NewChiRouter() *chi.Mux {
	return chi.NewRouter()
}

func NewHandler(s service.Service) *handlers.Handler {
	return &handlers.Handler{Service: s}
}

func NewService(db *gorm.DB) service.Service {
	return service.NewService(db)
}

func NewConnection() *gorm.DB {
	conn, err := db.InitDb()
	if err != nil {
		panic(err.Error())
	}
	return conn
}

func NewPoller(s service.Service, db *gorm.DB) *poller.Poller {
	return poller.New(s, db)
}
