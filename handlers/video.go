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
    }else if len(parts) == 1 {
        seconds, _ := strconv.Atoi(parts[1])
        totalSeconds = seconds
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

    formatRegex := regexp.MustCompile(`(?m)^\s*(\d+)\s+(\w+)\s+(audio only|video only|\d+x\d+|\d+p)\s+([a-zA-Z0-9\.\-\_]+)?\s*([\~\d\.]+(?:[kKM]i?B)?)?.*?\|\s+(.*?)$`)
    var formats []map[string]string

    for _, line := range lines {
        matches := formatRegex.FindStringSubmatch(line)
        if len(matches) > 0 {
            bitrate := extractBitrate(matches[6])

            formatType := "audio and video" 
            if matches[3] == "audio only" {
                formatType = "audio only"
            } else if matches[3] == "video only" {
                formatType = "video only"
            }

            formats = append(formats, map[string]string{
                "id":         matches[1],                 
                "extension":  matches[2],                 
                "label":      matches[3],                
                "codec":      strings.TrimSpace(matches[4]), 
                "bitrate":    bitrate,                   
                "formatType": formatType,                
                "additional": strings.TrimSpace(matches[6]), 
            })
        }
    }

    return formats
}

func extractBitrate(additional string) string {
    bitrateRegex := regexp.MustCompile(`([\~\d\.]+[kKM]i?B)`)
    match := bitrateRegex.FindStringSubmatch(additional)
    if len(match) > 0 {
        return match[1]
    }
    return "N/A" 
}


