package handlers

import (
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"ytclipper-go/jobs"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type CreateClipDTO struct {
    Url  string `json:"url" form:"url" validate:"required,url"`
    From string `json:"from" form:"from" validate:"required"`
    To   string `json:"to" form:"to" validate:"required"`
    Format string `json:"format" form:"format" validate:"required"`
}

// Helper function to validate YouTube URL
func isValidYoutubeUrl(url string) bool {
	regex := regexp.MustCompile(`http(?:s?):\/\/(?:www\.)?youtu(?:be\.com\/watch\?v=|\.be\/)([\w\-\_]*)(&(amp;)?‌​[\w\?‌​=]*)?`)
	return regex.MatchString(url)
}

// Helper function to validate time format (HH:MM:SS)
func isValidTimeFormat(time string) bool {
	regex := regexp.MustCompile(`^(?:[0-1]?[0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]$`)
	return regex.MatchString(time)
}

// Helper function to validate format (numeric)
func isValidFormat(format string) bool {
	regex := regexp.MustCompile(`^\d+$`)
	return regex.MatchString(format)
}

func CreateClip(c echo.Context) error {
    form := new(CreateClipDTO)
    if err := c.Bind(form); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
    }

	if !isValidYoutubeUrl(form.Url) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid YouTube URL"})
	}

	if !isValidTimeFormat(form.From) || !isValidTimeFormat(form.To) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid time format. Use HH:MM:SS."})
	}

	if !isValidFormat(form.Format) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid format. Must be a numeric value."})
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

    jobID := uuid.New().String()

    job := &jobs.Job{
        ID:     jobID,
        Status: jobs.StatusQueued,
    }
    jobs.JobsLock.Lock()
    jobs.Jobs[jobID] = job
    jobs.JobsLock.Unlock()

    go func(jobID string, createClipDTO *CreateClipDTO) {
        ProcessClip(jobID, createClipDTO.Url, createClipDTO.From, createClipDTO.To, createClipDTO.Format)
    }(jobID, form)

    return c.String(http.StatusCreated, jobID,)
}


func ProcessClip(jobID string, url string, from string, to string, selectedFormat string) {
    jobs.JobsLock.Lock()
    job := jobs.Jobs[jobID]
    job.Status = jobs.StatusProcessing
    jobs.JobsLock.Unlock()

    outputPath := filepath.Join("./videos", fmt.Sprintf("%s_clip.mp4", filepath.Base(jobID)))
    
    cmdArgs := []string{
        "-o", outputPath,
        "-f", selectedFormat,
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

    // Lock the job map and retrieve the job
    jobs.JobsLock.Lock()
    job, exists := jobs.Jobs[jobID]
    jobs.JobsLock.Unlock()

    if !exists || job.Status != jobs.StatusCompleted {
        return c.JSON(http.StatusNotFound, map[string]string{"error": "File not available"})
    }

    // Schedule file deletion after the response is sent
    // c.Response().After(func() {
    //     err := os.Remove(job.FilePath)
    //     if err != nil {
    //         // Log the error for debugging purposes
    //         c.Logger().Errorf("Failed to delete file %s: %v", job.FilePath, err)
    //     } else {
    //         // Optionally, clean up job data
    //         jobs.JobsLock.Lock()
    //         delete(jobs.Jobs, jobID)
    //         jobs.JobsLock.Unlock()
    //     }
    // })

    // Send the file to the client
    return c.File(job.FilePath)
}
