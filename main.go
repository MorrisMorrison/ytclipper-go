package main

import (
	"log"
	"os"
	"time"
	"ytclipper-go/config"
	custommiddleware "ytclipper-go/middleware"
	"ytclipper-go/routes"
	"ytclipper-go/scheduler"
	"ytclipper-go/utils"

	"github.com/MorrisMorrison/gutils/glogger"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

func checkDependencies() {
	glogger.Log.Info("Checking dependencies")

	if err := utils.CheckCommand("ffmpeg"); err != nil {
		log.Fatalf("Dependency check failed: %v", err)
	}

	if err := utils.CheckCommand("yt-dlp"); err != nil {
		log.Fatalf("Dependency check failed: %v", err)
	}

	glogger.Log.Info("All dependencies are installed.")
}

func setupEcho() {
	glogger.Log.Info("Setup echo")
	e := echo.New()

	glogger.Log.Infof("Debug enabled: %t", config.CONFIG.Debug)
	e.Debug = config.CONFIG.Debug
	e.Logger.SetOutput(os.Stdout)

	e.Static("/static", "static")

	e.Use(custommiddleware.BasicAuthMiddleware())

	routes.RegisterRoutes(e)

	glogger.Log.Infof("Starting server on port %s", config.CONFIG.Port)
	e.Use(middleware.Logger())
	limiterStore := middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(config.CONFIG.RateLimiterConfig.Rate),
		Burst:     config.CONFIG.RateLimiterConfig.Burst,
		ExpiresIn: time.Duration(config.CONFIG.RateLimiterConfig.ExpiresInMinutes) * time.Minute,
	})

	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store:   limiterStore,
	}))

	e.Logger.Fatal(e.Start(":" + config.CONFIG.Port))
}

func main() {
	glogger.Log.Info("Start ytclipper")
	checkDependencies()
	scheduler.StartClipCleanUpScheduler()
	scheduler.StartCookieMonitorScheduler()
	setupEcho()
}
