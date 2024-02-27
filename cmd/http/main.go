package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/nickquirk/life-dashboard-server/internal/handlers"
)

func main() {
	r := chi.NewRouter()
	handlers.GetRoutes(r)

	envs, err := godotenv.Read(".env")

	if err != nil {
		log.Fatal("Error reading .env file")
	}

	PORT := envs["PORT"]

	fmt.Printf("Listening on port %s\n", PORT)
	err = http.ListenAndServe(fmt.Sprintf(":%s", PORT), r)

	if err != nil {
		panic(err)
	}
}
