package api

import (
	"github.com/labstack/echo/v4"
)

// Register adds the health check and the JSON API to e. The page and the
// static files are served by the server package.
func Register(e *echo.Echo) {
	e.GET("/health", func(c echo.Context) error {
		return c.String(200, "Server is running")
	})

	e.POST("/api/v1/clip", CreateClip)
	e.GET("/api/v1/clip", GetClip)
	e.GET("/api/v1/jobs/status", GetJobStatus)
	e.GET("/api/v1/jobs/:id", GetJob)

	e.GET("/api/v1/video/info", GetVideoInfo)
}
