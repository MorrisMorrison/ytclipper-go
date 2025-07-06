package middleware

import (
	"crypto/subtle"
	"ytclipper-go/config"

	"github.com/MorrisMorrison/gutils/glogger"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// SetupBasicAuth configures basic authentication middleware for the Echo instance
// if credentials are provided via environment variables.
//
// Environment variables:
// - YTCLIPPER_BASIC_AUTH_USERNAME: Username for basic auth
// - YTCLIPPER_BASIC_AUTH_PASSWORD: Password for basic auth
//
// The middleware skips authentication for the /health endpoint to allow
// health checks to work without authentication.
func BasicAuthMiddleware() echo.MiddlewareFunc {
	username := config.CONFIG.BasicAuthConfig.Username
	password := config.CONFIG.BasicAuthConfig.Password

	// If no credentials configured, return a no-op middleware
	if username == "" || password == "" {
		glogger.Log.Info("Basic authentication disabled - no credentials configured")
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				return next(c)
			}
		}
	}

	glogger.Log.Info("Basic authentication enabled")

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
