package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func GetRoutes(mx *chi.Mux) {
	mx.Use(middleware.Logger)

	// Public Routes
	mx.Get("/", helloWorld)

	// Google OAuth2
	mx.Get("/google-login", GoogleLogin)
	mx.Get("/google-callback", GoogleCallback)

	// Private Routes
	mx.Group(func(r chi.Router) {
		r.Use(Authenticate)
		r.Get("/tasks", getAllTaskLists)
		r.Get("/tasks/{taskListId}", getAllTasksInList)
	})
}
