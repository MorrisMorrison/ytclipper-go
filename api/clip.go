package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"ytclipper-go/jobs"
	"ytclipper-go/videoprocessing"

	"github.com/labstack/echo/v4"
)

type CreateClipDTO struct {
    Url  string `json:"url" form:"url" validate:"required,url"`
    From string `json:"from" form:"from" validate:"required"`
    To   string `json:"to" form:"to" validate:"required"`
    Format string `json:"format" form:"format" validate:"required"`
}


func CreateClip(c echo.Context) error { 
    createClipDto := new(CreateClipDTO)
    if err := c.Bind(createClipDto); err != nil {
        rawJSON := new(strings.Builder)
        _, jsonErr := io.Copy(rawJSON, c.Request().Body)
        if jsonErr != nil {
            return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input: Could not read json body"})
        }

        return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Invalid input %s", rawJSON.String())})
    }

    if err := validateCreateClipDto(createClipDto); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
    }

    job := jobs.NewJob();

    go func(jobID string, createClipDTO *CreateClipDTO) {
        videoprocessing.ProcessClip(jobID, createClipDTO.Url, createClipDTO.From, createClipDTO.To, createClipDTO.Format)
    }(job.ID, createClipDto)

    return c.String(http.StatusCreated, job.ID,)
}


func GetClip(c echo.Context) error {
    jobID := c.QueryParam("jobId")
    job, exists := jobs.GetJobById(jobID)
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

    return c.File(job.FilePath)
}

