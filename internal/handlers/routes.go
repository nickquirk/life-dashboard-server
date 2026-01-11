package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func GetRoutes(mx *chi.Mux, h *Handler) {
	// Middleware
	mx.Use(middleware.Logger)
	mx.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any major browsers
	}))

	// Public Routes
	mx.Get("/", helloWorld)

	mx.Route("/api", func(r chi.Router) {
		// Public API routes

		// Google OAuth2
		mx.Get("/google-login", GoogleLogin)
		mx.Get("/google-callback", h.GoogleCallback)
		// Private API Routes (Authenticated)
		r.Group(func(auth chi.Router) {
			auth.Use(Authenticate)
			auth.Get("/auth/me", h.getCurrentUser)
			auth.Get("/tasks", h.getTaskLists)
			auth.Get("/tasks/{taskListId}", h.getTasksInList)
			auth.Get("/tasks/{taskListId}/active", h.getActiveTasksInList)
			auth.Patch("/tasks/{id}", h.updateTask)
		})
	})
}
