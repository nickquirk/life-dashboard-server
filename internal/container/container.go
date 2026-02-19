package container

import (
	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/config"
	"github.com/nickquirk/life-dashboard-server/internal/db"
	"github.com/nickquirk/life-dashboard-server/internal/handlers"
	"github.com/nickquirk/life-dashboard-server/internal/service"
	"go.uber.org/dig"
	"gorm.io/gorm"
)

func BuildContainer() *dig.Container {
	container := dig.New()

	container.Provide(config.LoadConfig)
	container.Provide(NewChiRouter)
	container.Provide(NewHandler)
	container.Provide(NewService)
	container.Provide(NewConnection)
	container.Provide(NewApplication)

	return container
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
