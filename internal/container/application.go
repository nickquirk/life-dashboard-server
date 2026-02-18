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
	"github.com/nickquirk/life-dashboard-server/internal/poller"
	"gorm.io/gorm"
)

// Application holds the "wired" dependencies of your system
type Application struct {
	Config config.Config
	Router *chi.Mux
	Poller *poller.Poller
	DB     *gorm.DB
}

// NewApplication is the constructor invoked by dig.
// It performs the "Wiring" phase (routes, configs).
func NewApplication(cfg config.Config, r *chi.Mux, h *handlers.Handler, gormDB *gorm.DB, p *poller.Poller) *Application {
	// Initialize routes (modifies the router in-place)
	handlers.GetRoutes(r, h)

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
	// Run Migrations (Fail fast if DB is down)
	log.Println("Running database migrations...")
	db.InitMigration(a.DB)

	// Configure HTTP Server
	port := a.Config.GetAsString("service.port")
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

	log.Println("Server exited properly")
	return nil
}
