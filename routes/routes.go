package routes

import (
	"ytclipper-go/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo) {
    e.GET("/", handlers.RenderHomePage)
    e.GET("/health", func(c echo.Context) error {
        return c.String(200, "Server is running")
    })

    e.POST("/api/v1/clip", handlers.DownloadAndSlice)
    e.GET("/api/v1/video/duration", handlers.GetVideoDuration)
}