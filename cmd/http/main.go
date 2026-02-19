package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/nickquirk/life-dashboard-server/internal/container"
)

func init() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Println("Warning: .env file not found, using system environment")
	}
}

func main() {
	c := container.BuildContainer()

	err := c.Invoke(func(app *container.Application) error {
		return app.Run()
	})

	if err != nil {
		log.Fatalf("Application failed to start: %v", err)
	}
}
