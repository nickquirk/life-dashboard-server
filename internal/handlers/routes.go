package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nickquirk/life-dashboard-server/internal/custommiddleware"
)

func GetRoutes(mx *chi.Mux) {
	mx.Use(middleware.Logger)

	// Public Routes
	mx.Get("/", helloWorld)
	mx.Post("/login", helloWorld)

	// Private Routes
	mx.Group(func(r chi.Router) {
		r.Use(custommiddleware.Authenticate)
		r.Get("/tasks", helloWorld)
	})
}
