package api

import (
	"net/http"
	"ytclipper-go/utils"
	"ytclipper-go/videoprocessing"

	"github.com/labstack/echo/v4"
)

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func GetVideoDuration(c echo.Context) error {
	url := c.QueryParam("youtubeUrl")
	if url == "" {
		c.Logger().Errorf("Url is required")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "URL is required"})
	}

	if !isValidYoutubeUrl(url) {
		c.Logger().Errorf("Invalid Youtube URL: %s", url)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid YouTube URL"})
	}

	// Extract user browser context
	userAgent := c.Request().UserAgent()
	cookies := c.Request().Header.Get("Cookie")

	// Check for YouTube-specific cookies sent via custom header
	youtubeCookies := c.Request().Header.Get("X-YouTube-Cookies")
	if youtubeCookies != "" {
		// Prefer YouTube-specific cookies if available
		cookies = youtubeCookies
	}

	duration, err := videoprocessing.GetVideoDurationWithContext(url, userAgent, cookies)
	if err != nil {

		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get video duration",
		})
	}

	if duration == "" {
		c.Logger().Error("Could not extract duration from yt-dlp output")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Could not extract duration from yt-dlp output",
		})
	}

	totalSeconds, err := utils.ToSeconds(duration)
	if err != nil {
		c.Logger().Error("Could not calculate total seconds")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Could not calculate total seconds",
		})
	}

	return c.JSON(http.StatusOK, totalSeconds)
}

func GetAvailableFormats(c echo.Context) error {
	url := c.QueryParam("youtubeUrl")
	if url == "" {
		c.Logger().Errorf("Url is required")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "URL is required"})
	}

	if !isValidYoutubeUrl(url) {
		c.Logger().Errorf("Invalid Youtube URL: %s", url)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid YouTube URL"})
	}

	// Extract user browser context
	userAgent := c.Request().UserAgent()
	cookies := c.Request().Header.Get("Cookie")

	// Check for YouTube-specific cookies sent via custom header
	youtubeCookies := c.Request().Header.Get("X-YouTube-Cookies")
	if youtubeCookies != "" {
		// Prefer YouTube-specific cookies if available
		cookies = youtubeCookies
		c.Logger().Infof("Using YouTube-specific cookies: %s", youtubeCookies[:minInt(100, len(youtubeCookies))])
	} else if cookies != "" {
		c.Logger().Infof("Using regular cookies: %s", cookies[:minInt(100, len(cookies))])
	} else {
		c.Logger().Infof("No cookies received")
	}

	formats, err := videoprocessing.GetAvailableFormatsWithContext(url, userAgent, cookies)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch formats",
		})
	}

	return c.JSON(http.StatusOK, formats)
}
