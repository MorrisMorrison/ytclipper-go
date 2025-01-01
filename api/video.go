package api

import (
	"net/http"
	"ytclipper-go/utils"
	"ytclipper-go/videoprocessing"

	"github.com/labstack/echo/v4"
)

func GetVideoDuration(c echo.Context) error {
    url := c.QueryParam("youtubeUrl")
    if url == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "URL is required"})
    }

    if !isValidYoutubeUrl(url) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid YouTube URL"})
	}

    duration, err := videoprocessing.GetVideoDuration(url) 
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error":   "Failed to get video duration",
        })
    }

    if duration == "" {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Could not extract duration from yt-dlp output",
        })
    }

    totalSeconds, err := utils.ToSeconds(duration); 
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Could not calculate total seconds",
        })
    }

    return c.JSON(http.StatusOK, totalSeconds)
}



func GetAvailableFormats(c echo.Context) error {
    url := c.QueryParam("youtubeUrl")
    if url == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "URL is required"})
    }

    if !isValidYoutubeUrl(url) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid YouTube URL"})
	}

    formats, err := videoprocessing.GetAvailableFormats(url)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error":   "Failed to fetch formats",
        })
    }

    return c.JSON(http.StatusOK, formats)
}




