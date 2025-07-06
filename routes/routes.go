package routes

import (
	"ytclipper-go/api"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo) {
	e.GET("/", api.RenderHomePage)
	e.GET("/health", func(c echo.Context) error {
		return c.String(200, "Server is running")
	})

	e.POST("/api/v1/clip", api.CreateClip)
	e.GET("/api/v1/clip", api.GetClip)
	e.GET("/api/v1/jobs/status", api.GetJobStatus)

	e.GET("/api/v1/video/duration", api.GetVideoDuration)
	e.GET("/api/v1/video/formats", api.GetAvailableFormats)
}
