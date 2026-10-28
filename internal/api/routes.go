package api

import (
	"net/http"

	"github.com/MorrisMorrison/ytclipper-go/internal/version"

	"github.com/labstack/echo/v4"
)

// Register adds the health check and the JSON API to e. The page and the
// static files are served by the server package.
func Register(e *echo.Echo) {
	e.GET("/health", Health)

	e.POST("/api/v1/clip", CreateClip)
	e.GET("/api/v1/clip", GetClip)
	e.GET("/api/v1/jobs/status", GetJobStatus)
	e.GET("/api/v1/jobs/:id", GetJob)

	e.GET("/api/v1/video/info", GetVideoInfo)
}

// Health answers "Server is running (<version>)". The text before the version
// stays the same, so monitors that match on it keep working.
func Health(c echo.Context) error {
	return c.String(http.StatusOK, "Server is running ("+version.Version+")")
}
