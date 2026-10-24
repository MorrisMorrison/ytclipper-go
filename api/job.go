package api

import (
	"fmt"
	"net/http"
	"net/url"
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

// JobDTO is a job as GET /api/v1/jobs/:id returns it. FileName and
// DownloadURL are set only once the job completed.
type JobDTO struct {
	ID          string         `json:"id"`
	Status      jobs.JobStatus `json:"status"`
	Error       string         `json:"error"`
	FileName    string         `json:"fileName,omitempty"`
	DownloadURL string         `json:"downloadUrl,omitempty"`
}

// GetJob returns a job's state as JSON.
func GetJob(c echo.Context) error {
	job, exists := jobs.GetJobById(c.Param("id"))
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Job not found"})
	}

	dto := JobDTO{ID: job.ID, Status: job.Status, Error: job.Error}
	if job.Status == jobs.StatusCompleted {
		dto.FileName = job.FileName
		dto.DownloadURL = "/api/v1/clip?jobId=" + url.QueryEscape(job.ID)
	}
	return c.JSON(http.StatusOK, dto)
}
