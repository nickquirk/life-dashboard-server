package container

import (
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/config"
	"github.com/nickquirk/life-dashboard-server/internal/crypto"
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
	container.Provide(NewTokenEncryptor)
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

func NewTokenEncryptor(cfg config.Config) crypto.TokenEncryptor {
	gcpProjectID := cfg.GetAsString("gcp.project_id")
	key, err := crypto.LoadEncryptionKey(gcpProjectID)
	if err != nil {
		log.Fatalf("Failed to load token encryption key: %v", err)
	}

	encryptor, err := crypto.NewAESGCMEncryptor(key)
	if err != nil {
		log.Fatalf("Failed to create token encryptor: %v", err)
	}
	return encryptor
}

func NewService(db *gorm.DB, encryptor crypto.TokenEncryptor) service.Service {
	return service.NewService(db, encryptor)
}

func NewConnection() *gorm.DB {
	conn, err := db.InitDb()
	if err != nil {
		panic(err.Error())
	}
	return conn
}
