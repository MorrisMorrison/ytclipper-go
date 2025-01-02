package main

import (
	"log"
	"time"
	"ytclipper-go/config"
	"ytclipper-go/routes"
	"ytclipper-go/utils"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
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
    appConfig := config.NewConfig()
    
    e.Debug = appConfig.Debug

    e.Static("/static", "static")
    routes.RegisterRoutes(e)

    log.Printf("Starting server on port %s", appConfig.Port)
    e.Use(middleware.Logger())
	limiterStore := middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(appConfig.RateLimiterConfig.Rate),              
		Burst:     appConfig.RateLimiterConfig.Burst,               
		ExpiresIn: time.Duration(appConfig.RateLimiterConfig.ExpiresInMinutes) * time.Minute, 
	})

	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper, 
		Store:   limiterStore,
	}))

    e.Logger.Fatal(e.Start(":" + appConfig.Port))
}

func main() {
    log.Println("Start ytclipper-go")
    checkDependencies();
    setupEcho();
}