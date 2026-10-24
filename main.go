package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
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

	for _, command := range []string{"ffmpeg", "yt-dlp"} {
		if err := utils.CheckCommand(command); err != nil {
			log.Fatalf("Dependency check failed: %v", err)
		}
	}

	glogger.Log.Info("All dependencies are installed.")
}

// redactProxy hides credentials in a proxy URL before it is logged.
func redactProxy(proxy string) string {
	u, err := url.Parse(proxy)
	if err != nil || u.User == nil {
		return proxy
	}
	u.User = url.User("***")
	return u.String()
}

func newServer() *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	glogger.Log.Infof("Debug enabled: %t", config.CONFIG.Debug)
	e.Debug = config.CONFIG.Debug
	e.Logger.SetOutput(os.Stdout)

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(custommiddleware.BasicAuthMiddleware())

	limiterStore := middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(config.CONFIG.RateLimiterConfig.Rate),
		Burst:     config.CONFIG.RateLimiterConfig.Burst,
		ExpiresIn: time.Duration(config.CONFIG.RateLimiterConfig.ExpiresInMinutes) * time.Minute,
	})
	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		// Only rate-limit the expensive endpoints, never static assets: a single
		// page load pulls several static files, which would otherwise burn the
		// per-IP budget (and break the e2e, where every test shares one IP).
		Skipper: func(c echo.Context) bool {
			return strings.HasPrefix(c.Request().URL.Path, "/static")
		},
		Store: limiterStore,
	}))

	e.Static("/static", "static")
	routes.RegisterRoutes(e)

	return e
}

func main() {
	glogger.Log.Info("Start ytclipper")
	checkDependencies()

	if proxy := config.CONFIG.YtDlpConfig.Proxy; proxy != "" {
		glogger.Log.Infof("yt-dlp proxy: %s", redactProxy(proxy))
	}

	scheduler.StartClipCleanUpScheduler()
	e := newServer()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		glogger.Log.Infof("Starting server on port %s", config.CONFIG.Port)
		if err := e.Start(":" + config.CONFIG.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Fatal(err)
		}
	}()

	<-ctx.Done()
	glogger.Log.Info("Shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatal(err)
	}
}
