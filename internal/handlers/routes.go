package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func GetRoutes(mx *chi.Mux) {
	// Middleware
	mx.Use(middleware.Logger)
	mx.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"http://localhost:3000"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	// Public Routes
	mx.Get("/", helloWorld)

	// Google OAuth2
	mx.Get("/google-login", GoogleLogin)
	mx.Get("/google-callback", GoogleCallback)

	// Private Routes
	mx.Group(func(r chi.Router) {
		r.Use(Authenticate)
		r.Get("/tasks", getTaskLists)
		r.Get("/tasks/{taskListId}", getTasksInList)
	})
}
