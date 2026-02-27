package handlers

import (
	"fmt"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func GetRoutes(mx *chi.Mux, h *Handler) error {
	clientUrl := os.Getenv("CLIENT_URL")
	if clientUrl == "" {
		return fmt.Errorf("CLIENT_URL environment variable is required but not set")
	}
	// Middleware
	mx.Use(middleware.Logger)
	mx.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{clientUrl},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any major browsers
	}))

	// Public Routes
	// Health
	mx.Get("/healthz", h.health)
	mx.Get("/readyz", h.ready)

	mx.Route("/api", func(r chi.Router) {
		// Google OAuth2
		r.Group(func(public chi.Router) {
			public.Get("/auth/google-login", h.googleLogin)
			public.Get("/auth/google-callback", h.googleCallback)
			public.Post("/auth/logout", h.logout)
			public.Post("/auth/refresh", h.refreshToken)
		})
		// Private API Routes (Authenticated)
		r.Group(func(auth chi.Router) {
			auth.Use(Authenticate)
			// auth
			auth.Get("/auth/me", h.getCurrentUser)
			// Task List
			auth.Get("/tasks", h.getTaskLists)
			auth.Post("/tasks/sync", h.syncTaskLists)
			// Tasks
			auth.Post("/tasks/{taskListId}", h.createTask)
			auth.Get("/tasks/{taskListId}", h.getTasksInList)
			auth.Post("/tasks/{taskListId}/sync", h.syncTasksInList)
			auth.Patch("/tasks/{id}", h.updateTask)
			auth.Delete("/tasks/{id}", h.deleteTask)
			// Calendar Events
			auth.Get("/calendar/events", h.getCalendarEvents)
			// Calendar Zones
			auth.Get("/zones", h.getZones)
			auth.Post("/zones", h.createZone)
			auth.Patch("/zones/{id}", h.updateZone)
			auth.Delete("/zones/{id}", h.deleteZone)
		})
	})
	return nil
}
