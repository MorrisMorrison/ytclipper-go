package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

type VideoForm struct {
    URL       string
    StartTime string
    EndTime   string
}

func RenderHomePage(c echo.Context) error {
    tmpl := template.Must(template.ParseFiles("templates/index.html"))
    return tmpl.Execute(c.Response().Writer, nil)
}

func DownloadAndSlice(c echo.Context) error {
    form := new(VideoForm)
    if err := c.Bind(form); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
    }

    outputPath := "./videos/clip.mp4"

    cmdArgs := []string{
        "-o", outputPath,
        "-f", "136",
        "-v",
        "--downloader", "ffmpeg",
        "--downloader-args", fmt.Sprintf("ffmpeg_i:-ss %s -to %s", form.StartTime, form.EndTime),
        form.URL,
    }


    cmd := exec.Command("yt-dlp", cmdArgs...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error":   "Failed to download and slice video",
            "details": string(output),
        })
    }

    return c.JSON(http.StatusOK, map[string]string{
        "message":  "Video downloaded and sliced successfully",
        "filePath": outputPath,
    })
}

func GetVideoDuration(c echo.Context) error {
    url := c.QueryParam("youtubeUrl")
    if url == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "URL is required"})
    }

    cmdArgs := []string{
        "--get-duration",
        "--no-warnings", // Suppress warnings
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

    // Parse duration (e.g., "12:40" or "01:12:40")
    parts := strings.Split(durationStr, ":")
    var totalSeconds int

    if len(parts) == 3 {
        // Format: HH:MM:SS
        hours, _ := strconv.Atoi(parts[0])
        minutes, _ := strconv.Atoi(parts[1])
        seconds, _ := strconv.Atoi(parts[2])
        totalSeconds = hours*3600 + minutes*60 + seconds
    } else if len(parts) == 2 {
        // Format: MM:SS
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
