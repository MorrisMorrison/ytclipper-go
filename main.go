package main

import (
	"log"
	"os"
	"ytclipper-go/routes"

	"github.com/labstack/echo/v4"
)

func main() {
    e := echo.New()

	e.Static("/static", "static")

    routes.RegisterRoutes(e)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    log.Printf("Starting server on port %s", port)
    e.Logger.Fatal(e.Start(":" + port))
}