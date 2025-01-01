package api

import (
	"net/http"
	"ytclipper-go/jobs"

	"github.com/labstack/echo/v4"
)

func GetJobStatus(c echo.Context) error {
    jobID := c.QueryParam("jobId")

    job, exists := jobs.GetJobById(jobID)
    if !exists {
        return c.JSON(http.StatusNotFound, map[string]string{"error": "Job not found"})
    }

    if (job.Status== jobs.StatusProcessing){
        return c.JSON(http.StatusCreated, nil)
    } else if (job.Status == jobs.StatusCompleted){
        return c.JSON(http.StatusOK, job.FilePath)        
    }

    return c.JSON(http.StatusOK, job)
}