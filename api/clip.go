package api

import (
	"net/http"
	"path/filepath"
	"ytclipper-go/jobs"
	"ytclipper-go/videoprocessing"

	"github.com/labstack/echo/v4"
)

// CreateClipDTO is the clip request. Exactly one of Preset ("video" or
// "audio") and Format (a yt-dlp format id such as "137" or "137+140") is set.
type CreateClipDTO struct {
	Url    string `json:"url" form:"url"`
	From   string `json:"from" form:"from"`
	To     string `json:"to" form:"to"`
	Preset string `json:"preset" form:"preset"`
	Format string `json:"format" form:"format"`
}

func CreateClip(c echo.Context) error {
	createClipDto := new(CreateClipDTO)
	if err := c.Bind(createClipDto); err != nil {
		c.Logger().Errorf("Invalid input: could not bind request: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	req, err := validateCreateClipDto(createClipDto)
	if err != nil {
		c.Logger().Errorf("Invalid DTO: %s", err.Error())
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	job := jobs.NewJob()
	go videoprocessing.ProcessClip(job.ID, req)

	return c.String(http.StatusCreated, job.ID)
}

func GetClip(c echo.Context) error {
	jobID := c.QueryParam("jobId")
	job, exists := jobs.GetJobById(jobID)
	if !exists || job.Status != jobs.StatusCompleted {
		c.Logger().Errorf("Job does not exist or is not completed. JobId:%s", jobID)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Job does not exist"})
	}

	name := job.FileName
	if name == "" {
		name = filepath.Base(job.FilePath)
	}
	return c.Attachment(job.FilePath, name)
}
