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

func GetAvailableFormats(c echo.Context) error {
    url := c.QueryParam("youtubeUrl")
    if url == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "URL is required"})
    }

    cmd := exec.Command("yt-dlp", "-F", url)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error":   "Failed to fetch formats",
            "details": string(output),
        })
    }

    formats := parseFormats(string(output))
    return c.JSON(http.StatusOK, formats)
}

func parseFormats(output string) []map[string]string {
    lines := strings.Split(output, "\n")

    formatRegex := regexp.MustCompile(`(?m)^\s*(\d+)\s+(\w+)\s+(\d+x\d+|\d+p|audio only)\s+([a-zA-Z0-9\.\-\_]+)\s+([\d\.]+[a-zA-Z]+)?\s+.*$`)
    var formats []map[string]string

    for _, line := range lines {
        matches := formatRegex.FindStringSubmatch(line)
        if len(matches) > 0 {
            formats = append(formats, map[string]string{
                "id":        matches[1], // Format ID
                "extension": matches[2], // File extension (e.g., mp4)
                "label":     matches[3], // Resolution or audio-only
                "codec":     matches[4], // Codec (e.g., avc1, vp9)
                "bitrate":   matches[5], // Bitrate (e.g., 128k, 1.5M)
            })
        }
    }

    return formats
}
