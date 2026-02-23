package main

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/nickquirk/life-dashboard-server/internal/container"
)

func init() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if os.Getenv("ENV") != "prod" {
		err := godotenv.Load(".env")
		if err != nil {
			slog.Warn("env file not found, using system environment")
		}
	}
}

func main() {
	c := container.BuildContainer()

	err := c.Invoke(func(app *container.Application) error {
		return app.Run()
	})

	if err != nil {
		slog.Error("application failed to start", "error", err)
		os.Exit(1)
	}
}
