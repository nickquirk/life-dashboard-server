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
	"gorm.io/gorm"
)

type Application struct {
	Config config.Config
	Router *chi.Mux
	DB     *gorm.DB
}

// NewApplication is the constructor invoked by dig.
// It performs the "Wiring" phase (routes, configs).
func NewApplication(cfg config.Config, r *chi.Mux, h *handlers.Handler, gormDB *gorm.DB) *Application {
	// Initialize routes (modifies the router in-place)
	if err := handlers.GetRoutes(r, h); err != nil {
		log.Fatalf("Failed to initialize routes: %v", err)
	}

	// Initialize Google Config
	config.GetGoogleConfig()

	return &Application{
		Config: cfg,
		Router: r,
		DB:     gormDB,
	}
}

// Run executes the "Startup" phase (migrations, server, background jobs)
// and handles the "Shutdown" phase.
func (a *Application) Run() error {
	// Check if we are running as a dedicated Migration Job
	if os.Getenv("MIGRATE_ONLY") == "true" {
		log.Println("Job mode detected: Running database migrations...")
		db.InitMigration(a.DB)
		log.Println("Migrations complete. Exiting gracefully.")
		return nil // Exit the application, do NOT start the server
	}

	// If we are not in prod auto-migrate on startup
	if os.Getenv("ENV") != "prod" {
		log.Println("Local mode detected: Running database migrations...")
		db.InitMigration(a.DB)
	} else {
		log.Println("Prod mode detected: Skipping startup migrations.")
	}

	// Configure HTTP Server
	port := os.Getenv("PORT")
	if port == "" {
		// falback to config
		port = a.Config.GetAsString("service.port")
	}
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

	// Graceful Shutdown Sequence
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server exited gracefully")
	return nil
}
