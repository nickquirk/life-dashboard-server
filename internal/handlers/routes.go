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
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any major browsers
	}))

	// Public Routes mx is router route
	mx.Get("/", helloWorld)
	mx.Route("/api", func(r chi.Router) {
		// Public API routes

		// Google OAuth2
		r.Group(func(public chi.Router) {
			public.Get("/auth/google-login", GoogleLogin)
			public.Get("/auth/google-callback", h.GoogleCallback)
			public.Post("/auth/logout", h.logout)
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
		})
	})
}
