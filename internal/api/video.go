package api

import (
	"net/http"

	"github.com/MorrisMorrison/ytclipper-go/internal/videoprocessing"

	"github.com/labstack/echo/v4"
)

// GetVideoInfo returns title, duration, thumbnail and formats of the video in
// the youtubeUrl query parameter (cached per video ID).
func GetVideoInfo(c echo.Context) error {
	rawURL := c.QueryParam("youtubeUrl")
	if rawURL == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "URL is required"})
	}

	videoID, ok := extractVideoID(rawURL)
	if !ok {
		c.Logger().Errorf("Invalid Youtube URL: %s", rawURL)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid YouTube URL"})
	}

	info, err := videoprocessing.GetVideoInfo(videoID)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, info)
}
