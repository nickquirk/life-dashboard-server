package handlers

import (
	"net/http"

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
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any major browsers
	}))

	// Public Routes
	mx.Get("/", helloWorld)
	mx.Post("/users", func(w http.ResponseWriter, r *http.Request) {
		// h.CreateUser(w, r)
	})

	// Google OAuth2
	mx.Get("/google-login", GoogleLogin)
	mx.Get("/google-callback", h.GoogleCallback)

	// Private Routes
	mx.Group(func(r chi.Router) {
		r.Use(Authenticate)
		r.Get("/tasks", getTaskLists)
		r.Get("/tasks/{taskListId}", getTasksInList)
	})
}
