package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nickquirk/life-dashboard-server/internal/handlers"
)

const PORT = "4000"

func main() {
	r := chi.NewRouter()
	handlers.GetRoutes(r)

	fmt.Printf("Listening on port %s\n", PORT)
	err := http.ListenAndServe(fmt.Sprintf(":%s", PORT), r)

	if err != nil {
		panic(err)
	}
}
