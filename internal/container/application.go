package container

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/config"
	"github.com/nickquirk/life-dashboard-server/internal/db"
	"github.com/nickquirk/life-dashboard-server/internal/handlers"
	"github.com/nickquirk/life-dashboard-server/internal/helper"
	"github.com/nickquirk/life-dashboard-server/internal/poller"
	"gorm.io/gorm"
)

// Application holds the "wired" dependencies of your system
type Application struct {
	App    helper.App
	Router *chi.Mux
	Poller *poller.Poller
	DB     *gorm.DB
}

// NewApplication is the constructor invoked by dig.
// It performs the "Wiring" phase (routes, configs).
func NewApplication(app helper.App, r *chi.Mux, h *handlers.Handler, gormDB *gorm.DB, p *poller.Poller) *Application {
	// Initialize routes (modifies the router in-place)
	handlers.GetRoutes(r, h)

	// Initialize Google Config
	config.GetGoogleConfig()

	return &Application{
		App:    app,
		Router: r,
		Poller: p,
		DB:     gormDB,
	}
}

// Run executes the "Startup" phase (migrations, server, background jobs)
// and handles the "Shutdown" phase.
func (a *Application) Run() error {
	// Run Migrations (Fail fast if DB is down)
	log.Println("Running database migrations...")
	db.InitMigration(a.DB)

	// Start Background Workers
	a.Poller.Start()

	// Configure HTTP Server
	port := a.App.GetConfig().GetAsString("service.port")
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      a.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Start Server in Goroutine
	go func() {
		log.Printf("Server listening on port %s\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Block and Wait for Shutdown Signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")

	// Graceful Shutdown Sequence
	a.Poller.Stop() // Stop accepting new background work

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server exited properly")
	return nil
}
