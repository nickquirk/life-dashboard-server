package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/nickquirk/life-dashboard-server/internal/config"
	"github.com/nickquirk/life-dashboard-server/internal/handlers"
)

func init() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	r := chi.NewRouter()
	handlers.GetRoutes(r)

	config.GoogleConfig()

	PORT := os.Getenv("PORT")

	log.Printf("Listening on port %s\n", PORT)
	err := http.ListenAndServe(fmt.Sprintf(":%s", PORT), r)

	if err != nil {
		panic(err)
	}
}
