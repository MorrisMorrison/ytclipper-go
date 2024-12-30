package main

import (
	"log"
	"os"
	"time"
	"ytclipper-go/routes"
	"ytclipper-go/utils"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func checkDependencies(){
   log.Println("Checking dependencies...")

    if err := utils.CheckCommand("ffmpeg"); err != nil {
        log.Fatalf("Dependency check failed: %v", err)
    }

    if err := utils.CheckCommand("yt-dlp"); err != nil {
        log.Fatalf("Dependency check failed: %v", err)
    }

    log.Println("All dependencies are installed.")
}

func setupEcho(){
    e := echo.New()

    e.Debug = true


    e.Static("/static", "static")

    e.Logger.Info("setting up routes ...")
    routes.RegisterRoutes(e)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("Starting server on port %s", port)

    e.Use(middleware.Logger())
	limiterStore := middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      5,              
		Burst:     20,               
		ExpiresIn: 1 * time.Minute, 
	})

	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper, 
		Store:   limiterStore,
	}))

    e.Logger.Fatal(e.Start(":" + port))
}

func main() {
    log.Println("Start ytclipper-go")
    checkDependencies();
    setupEcho();
}