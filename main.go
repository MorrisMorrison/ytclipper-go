package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
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

// isSocksProxy reports whether yt-dlp's proxy is a SOCKS proxy. yt-dlp can use
// it, but ffmpeg (which downloads --download-sections) cannot.
func isSocksProxy(proxy string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(proxy)), "socks")
}

// newIPExtractor decides where the client IP (used by the rate limiter and
// the logs) comes from. Without trusted proxies it is the connection's remote
// address, so a client cannot pick its own rate limit bucket by sending
// X-Forwarded-For. With trusted proxies (IPs or CIDRs), X-Forwarded-For is
// followed back through those proxies, and only those.
func newIPExtractor(trustedProxies []string) (echo.IPExtractor, error) {
	if len(trustedProxies) == 0 {
		return echo.ExtractIPDirect(), nil
	}

	options := []echo.TrustOption{
		// Trust exactly the configured proxies, not echo's defaults
		// (loopback, link-local and private networks).
		echo.TrustLoopback(false),
		echo.TrustLinkLocal(false),
		echo.TrustPrivateNet(false),
	}
	for _, p := range trustedProxies {
		ipNet, err := parseIPOrCIDR(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", config.CONFIG_KEY_TRUSTED_PROXIES, err)
		}
		options = append(options, echo.TrustIPRange(ipNet))
	}
	return echo.ExtractIPFromXFFHeader(options...), nil
}

func parseIPOrCIDR(s string) (*net.IPNet, error) {
	if strings.Contains(s, "/") {
		_, ipNet, err := net.ParseCIDR(s)
		return ipNet, err
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP or CIDR %q", s)
	}
	bits := 128
	if ip4 := ip.To4(); ip4 != nil {
		ip, bits = ip4, 32
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}, nil
}

func newServer() (*echo.Echo, error) {
	e := echo.New()
	e.HideBanner = true

	glogger.Log.Infof("Debug enabled: %t", config.CONFIG.Debug)
	e.Debug = config.CONFIG.Debug
	e.Logger.SetOutput(os.Stdout)

	ipExtractor, err := newIPExtractor(config.CONFIG.TrustedProxies)
	if err != nil {
		return nil, err
	}
	e.IPExtractor = ipExtractor
	if len(config.CONFIG.TrustedProxies) > 0 {
		glogger.Log.Infof("Trusted proxies: %s", strings.Join(config.CONFIG.TrustedProxies, ", "))
	}

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// The rate limiter runs before basic auth, so failed logins are limited
	// too.
	limiterStore := middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(config.CONFIG.RateLimiterConfig.Rate),
		Burst:     config.CONFIG.RateLimiterConfig.Burst,
		ExpiresIn: time.Duration(config.CONFIG.RateLimiterConfig.ExpiresInMinutes) * time.Minute,
	})
	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		// Only rate-limit the expensive endpoints, never static assets: a single
		// page load pulls several static files, which would otherwise burn the
		// per-IP budget (and break the e2e, where every test shares one IP).
		// Health checks are not limited either.
		Skipper: func(c echo.Context) bool {
			path := c.Request().URL.Path
			return strings.HasPrefix(path, "/static") || path == "/health"
		},
		Store: limiterStore,
	}))
	e.Use(custommiddleware.BasicAuthMiddleware())

	e.Static("/static", "static")
	routes.RegisterRoutes(e)

	return e, nil
}

func main() {
	glogger.Log.Info("Start ytclipper")
	checkDependencies()

	if proxy := config.CONFIG.YtDlpConfig.Proxy; proxy != "" {
		glogger.Log.Infof("yt-dlp proxy: %s", redactProxy(proxy))
		if isSocksProxy(proxy) {
			glogger.Log.Warning("yt-dlp proxy is SOCKS: clips are downloaded by ffmpeg, which can't use SOCKS, " +
				"so YouTube will usually answer 403. Use an HTTP proxy (http://host:port) instead.")
		}
	}

	scheduler.StartClipCleanUpScheduler()
	e, err := newServer()
	if err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

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
