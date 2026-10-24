package api

import (
	"fmt"
	"net/http"
	"ytclipper-go/jobs"

	"github.com/labstack/echo/v4"
)

// GetJobStatus reports a job's state through the status code the page polls
// on: 201 while queued or processing, 200 when done, 500 on failure.
func GetJobStatus(c echo.Context) error {
	jobID := c.QueryParam("jobId")

	job, exists := jobs.GetJobById(jobID)
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": fmt.Sprintf("Job %s not found", jobID)})
	}

	switch job.Status {
	case jobs.StatusQueued, jobs.StatusProcessing:
		return c.JSON(http.StatusCreated, nil)
	case jobs.StatusCompleted:
		return c.JSON(http.StatusOK, nil)
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": job.Error})
	}
}
