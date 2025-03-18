package api

import (
	"fmt"
	"net/http"
	"ytclipper-go/jobs"

	"github.com/labstack/echo/v4"
)

func GetJobStatus(c echo.Context) error {
    jobID := c.QueryParam("jobId")

    job, exists := jobs.GetJobById(jobID)
    if !exists {
        return c.JSON(http.StatusNotFound, map[string]string{"error": fmt.Sprintf("Job %s not found", jobID)})
    }

    switch job.Status {
    case jobs.StatusQueued:
    case jobs.StatusProcessing:
        return c.JSON(http.StatusProcessing, nil)
    case jobs.StatusCompleted:
        return c.JSON(http.StatusOK, job.FilePath)
    case jobs.StatusError:
    default:
        return c.JSON(http.StatusInternalServerError, job.Error)
    }   

    return c.JSON(http.StatusInternalServerError, job.Error)
}