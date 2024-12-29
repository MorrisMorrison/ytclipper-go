package handlers

import (
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

func GetVideoDuration(c echo.Context) error {
    url := c.QueryParam("youtubeUrl")
    if url == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "URL is required"})
    }

    cmdArgs := []string{
        "--get-duration",
        "--no-warnings", 
        url,
    }
    cmd := exec.Command("yt-dlp", cmdArgs...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error":   "Failed to get video duration",
            "details": string(output),
        })
    }

    fmt.Printf("Raw output from yt-dlp: %s\n", string(output))

    durationStr := ExtractDuration(string(output))
    if durationStr == "" {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Could not extract duration from yt-dlp output",
            "details": string(output),
        })
    }

    parts := strings.Split(durationStr, ":")
    var totalSeconds int

    if len(parts) == 3 {
        hours, _ := strconv.Atoi(parts[0])
        minutes, _ := strconv.Atoi(parts[1])
        seconds, _ := strconv.Atoi(parts[2])
        totalSeconds = hours*3600 + minutes*60 + seconds
    } else if len(parts) == 2 {
        minutes, _ := strconv.Atoi(parts[0])
        seconds, _ := strconv.Atoi(parts[1])
        totalSeconds = minutes*60 + seconds
    } else {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Invalid duration format received",
            "details": durationStr,
        })
    }


    return c.JSON(http.StatusOK, totalSeconds)
}

func ExtractDuration(output string) string {
    re := regexp.MustCompile(`\d+:\d{2}(?::\d{2})?`)
    matches := re.FindStringSubmatch(output)
    if len(matches) > 0 {
        return matches[0]
    }
    return ""
}
