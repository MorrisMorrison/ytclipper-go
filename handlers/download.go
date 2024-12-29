package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"ytclipper-go/jobmanager"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type VideoForm struct {
    Url  string `json:"url" form:"url" validate:"required,url"`
    From string `json:"from" form:"from" validate:"required"`
    To   string `json:"to" form:"to" validate:"required"`
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

    cmdArgs := []string{"-F", form.Url}
    cmd := exec.Command("yt-dlp", cmdArgs...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error":   "Failed to fetch available formats",
            "details": string(output),
        })
    }

    selectedFormat, parseErr := selectMidQualityFormat(string(output))
    if parseErr != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error":   "Failed to select a suitable format",
            "details": parseErr.Error(),
        })
    }


    jobID := uuid.New().String()

    job := &jobmanager.Job{
        ID:     jobID,
        Status: jobmanager.StatusQueued,
    }
    jobmanager.JobsLock.Lock()
    jobmanager.Jobs[jobID] = job
    jobmanager.JobsLock.Unlock()

    go func(jobID string, form *VideoForm) {
        processDownloadAndSlice(jobID, form, selectedFormat)
    }(jobID, form)

    return c.String(http.StatusCreated, jobID,)
}


func processDownloadAndSlice(jobID string, form *VideoForm, selectedFormat string) {
    jobmanager.JobsLock.Lock()
    job := jobmanager.Jobs[jobID]
    job.Status = jobmanager.StatusProcessing
    jobmanager.JobsLock.Unlock()

    outputPath := fmt.Sprintf("./videos/%s_clip.mp4", jobID)
    
    cmdArgs := []string{
        "-o", outputPath,
        "-f", "298",
        "-v",
        "--downloader", "ffmpeg",
        "--downloader-args", fmt.Sprintf("ffmpeg_i:-ss %s -to %s", form.From, form.To),
        form.Url,
    }
    cmd := exec.Command("yt-dlp", cmdArgs...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        jobmanager.JobsLock.Lock()
        job.Status = jobmanager.StatusError
        job.Error = fmt.Sprintf("Failed to download video: %s", string(output))
        jobmanager.JobsLock.Unlock()
        return
    }

    // sliceCmd := exec.Command("ffmpeg", "-i", outputPath, "-ss", form.From, "-to", form.To, "-c", "copy", slicedPath)
    // sliceOutput, err := sliceCmd.CombinedOutput()
    // if err != nil {
    //     jobmanager.JobsLock.Lock()
    //     job.Status = jobmanager.StatusError
    //     job.Error = fmt.Sprintf("Failed to slice video locally: %s", string(sliceOutput))
    //     jobmanager.JobsLock.Unlock()
    //     return
    // }

    jobmanager.JobsLock.Lock()
    job.Status = jobmanager.StatusCompleted
    job.FilePath = outputPath
    jobmanager.JobsLock.Unlock()
}

func selectMidQualityFormat(output string) (string, error) {
    lines := strings.Split(output, "\n")
    var formats []map[string]string

    formatRegex := regexp.MustCompile(`(?m)^\s*(\d+)\s+(\w+)\s+(\d+x\d+|\d+p|audio only)\s+(.*)$`)

    for _, line := range lines {
        matches := formatRegex.FindStringSubmatch(line)
        if len(matches) > 0 {
            formats = append(formats, map[string]string{
                "id":       matches[1], 
                "ext":      matches[2], 
                "quality":  matches[3], 
                "features": matches[4], 
            })
        }
    }

    var selectedFormat string
    for _, format := range formats {
        if strings.Contains(format["features"], "audio") && format["ext"] == "mp4" {
            selectedFormat = format["id"]
            break
        }
    }

    if selectedFormat == "" {
        return "", fmt.Errorf("no suitable format found")
    }

    return selectedFormat, nil
}

func GetJobStatus(c echo.Context) error {
    jobID := c.QueryParam("jobId")

    jobmanager.JobsLock.Lock()
    job, exists := jobmanager.Jobs[jobID]
    jobmanager.JobsLock.Unlock()

    if !exists {
        return c.JSON(http.StatusNotFound, map[string]string{"error": "Job not found"})
    }

    if (job.Status== jobmanager.StatusProcessing){
        return c.JSON(http.StatusCreated, nil)
    } else if (job.Status == jobmanager.StatusCompleted){
        return c.JSON(http.StatusOK, job.FilePath)        
    }

    return c.JSON(http.StatusOK, job)
}


func DownloadFile(c echo.Context) error {
    jobID := c.QueryParam("jobId")

    jobmanager.JobsLock.Lock()
    job, exists := jobmanager.Jobs[jobID]
    jobmanager.JobsLock.Unlock()

    if !exists || job.Status != jobmanager.StatusCompleted {
        return c.JSON(http.StatusNotFound, map[string]string{"error": "File not available"})
    }

    return c.File(job.FilePath)
}

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
