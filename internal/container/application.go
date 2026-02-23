package container

import (
	"context"
	"fmt"
	"log/slog"
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
		slog.Error("failed to initialize routes", "error", err)
		os.Exit(1)
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
		slog.Info("running database migrations", "mode", "job")
		db.InitMigration(a.DB)
		slog.Info("migrations complete, exiting")
		return nil // Exit the application, do NOT start the server
	}

	// If we are not in prod auto-migrate on startup
	if os.Getenv("ENV") != "prod" {
		slog.Info("running database migrations", "mode", "local")
		db.InitMigration(a.DB)
	} else {
		slog.Info("skipping startup migrations", "mode", "prod")
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
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start Server in Goroutine
	go func() {
		slog.Info("server listening", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Block and Wait for Shutdown Signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful Shutdown Sequence
	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	slog.Info("server exited gracefully")
	return nil
}
