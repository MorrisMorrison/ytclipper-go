package main

import (
	"log"
	"os"
	"ytclipper-go/routes"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
    e := echo.New()

    // Enable debug mode
    e.Debug = true

    e.Logger.Info("start ytclipper-go")

    // Serve static files
    e.Static("/static", "static")

    e.Logger.Info("setting up routes ...")
    routes.RegisterRoutes(e)

    // Define port
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("Starting server on port %s", port)

    // Add request logging middleware
    e.Use(middleware.Logger())

    // Start the server
    e.Logger.Fatal(e.Start(":" + port))
}