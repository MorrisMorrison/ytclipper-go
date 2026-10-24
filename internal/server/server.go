// Package server builds the Echo server: middleware, the page and static
// files, and the API routes.
package server

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/MorrisMorrison/ytclipper-go/internal/api"
	"github.com/MorrisMorrison/ytclipper-go/internal/config"

	"github.com/MorrisMorrison/gutils/glogger"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

// New builds the server from config.CONFIG.
func New() (*echo.Echo, error) {
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
	e.Use(basicAuth())

	if err := registerWeb(e, config.CONFIG.DevAssetsDir); err != nil {
		return nil, err
	}
	api.Register(e)

	return e, nil
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
