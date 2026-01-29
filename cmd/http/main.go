package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/nickquirk/life-dashboard-server/internal/config"
	"github.com/nickquirk/life-dashboard-server/internal/container"
	"github.com/nickquirk/life-dashboard-server/internal/db"
	"github.com/nickquirk/life-dashboard-server/internal/handlers"
	"github.com/nickquirk/life-dashboard-server/internal/helper"
	"github.com/nickquirk/life-dashboard-server/internal/poller"
	"github.com/nickquirk/life-dashboard-server/internal/service"
	"gorm.io/gorm"
)

func init() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	container := container.BuildContainer()

	err := container.Invoke(func(app helper.App, r *chi.Mux, h *handlers.Handler, s service.Service, gormDB *gorm.DB, p *poller.Poller) {

		db.InitMigration(gormDB)
		handlers.GetRoutes(r, h)
		config.GetGoogleConfig()
		p.Start()

		PORT := app.GetConfig().GetAsString("service.port")

		log.Printf("Listening on port %s\n", PORT)
		err := http.ListenAndServe(fmt.Sprintf(":%s", PORT), r)

		if err != nil {
			panic(err)
		}
	})

	if err != nil {
		panic(err)
	}
}
