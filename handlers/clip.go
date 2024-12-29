package handlers

import (
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"ytclipper-go/jobs"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type CreateClipDTO struct {
    Url  string `json:"url" form:"url" validate:"required,url"`
    From string `json:"from" form:"from" validate:"required"`
    To   string `json:"to" form:"to" validate:"required"`
}

func CreateClip(c echo.Context) error {
    form := new(CreateClipDTO)
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

    selectedFormat, parseErr := selectAvailableMp4FormatIncludingAudio(string(output))
    if parseErr != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{
            "error":   "Failed to select a suitable format",
            "details": parseErr.Error(),
        })
    }


    jobID := uuid.New().String()

    job := &jobs.Job{
        ID:     jobID,
        Status: jobs.StatusQueued,
    }
    jobs.JobsLock.Lock()
    jobs.Jobs[jobID] = job
    jobs.JobsLock.Unlock()

    go func(jobID string, createClipDTO *CreateClipDTO) {
        ProcessClip(jobID, createClipDTO.Url, createClipDTO.From, createClipDTO.To, selectedFormat)
    }(jobID, form)

    return c.String(http.StatusCreated, jobID,)
}


func ProcessClip(jobID string, url string, from string, to string, selectedFormat string) {
    jobs.JobsLock.Lock()
    job := jobs.Jobs[jobID]
    job.Status = jobs.StatusProcessing
    jobs.JobsLock.Unlock()

    outputPath := fmt.Sprintf("./videos/%s_clip.mp4", jobID)
    
    cmdArgs := []string{
        "-o", outputPath,
        "-f", "298",
        "-v",
        "--downloader", "ffmpeg",
        "--downloader-args", fmt.Sprintf("ffmpeg_i:-ss %s -to %s", from, to),
        url,
    }
    cmd := exec.Command("yt-dlp", cmdArgs...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        jobs.JobsLock.Lock()
        job.Status = jobs.StatusError
        job.Error = fmt.Sprintf("Failed to download video: %s", string(output))
        jobs.JobsLock.Unlock()
        return
    }

    // sliceCmd := exec.Command("ffmpeg", "-i", outputPath, "-ss", form.From, "-to", form.To, "-c", "copy", slicedPath)
    // sliceOutput, err := sliceCmd.CombinedOutput()
    // if err != nil {
    //     jobs.JobsLock.Lock()
    //     job.Status = jobs.StatusError
    //     job.Error = fmt.Sprintf("Failed to slice video locally: %s", string(sliceOutput))
    //     jobs.JobsLock.Unlock()
    //     return
    // }

    jobs.JobsLock.Lock()
    job.Status = jobs.StatusCompleted
    job.FilePath = outputPath
    jobs.JobsLock.Unlock()
}


func GetClip(c echo.Context) error {
    jobID := c.QueryParam("jobId")

    jobs.JobsLock.Lock()
    job, exists := jobs.Jobs[jobID]
    jobs.JobsLock.Unlock()

    if !exists || job.Status != jobs.StatusCompleted {
        return c.JSON(http.StatusNotFound, map[string]string{"error": "File not available"})
    }

    return c.File(job.FilePath)
}


func selectAvailableMp4FormatIncludingAudio(output string) (string, error) {
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