package server

import (
	"crypto/subtle"
	"log/slog"

	"github.com/MorrisMorrison/ytclipper-go/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// basicAuth returns basic authentication middleware
// if credentials are provided via environment variables.
//
// Environment variables:
// - YTCLIPPER_AUTH_USERNAME: Username for basic auth
// - YTCLIPPER_AUTH_PASSWORD: Password for basic auth
//
// The middleware skips authentication for the /health endpoint to allow
// health checks to work without authentication.
func basicAuth() echo.MiddlewareFunc {
	username := config.CONFIG.BasicAuthConfig.Username
	password := config.CONFIG.BasicAuthConfig.Password

	// If no credentials configured, return a no-op middleware
	if username == "" || password == "" {
		slog.Info("Basic authentication disabled: no credentials configured")
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				return next(c)
			}
		}
	}

	slog.Info("Basic authentication enabled")

	// Create the actual auth middleware
	return middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
		Validator: func(u, p string, c echo.Context) (bool, error) {
			return subtle.ConstantTimeCompare([]byte(u), []byte(username)) == 1 &&
				subtle.ConstantTimeCompare([]byte(p), []byte(password)) == 1, nil
		},
		Skipper: func(c echo.Context) bool {
			return c.Path() == "/health"
		},
		Realm: "YTClipper",
	})
}
