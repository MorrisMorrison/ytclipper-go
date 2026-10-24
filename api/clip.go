package api

import (
	"net/http"
	"path/filepath"
	"ytclipper-go/jobs"
	"ytclipper-go/videoprocessing"

	"github.com/labstack/echo/v4"
)

type CreateClipDTO struct {
	Url    string `json:"url" form:"url"`
	From   string `json:"from" form:"from"`
	To     string `json:"to" form:"to"`
	Format string `json:"format" form:"format"`
}

func CreateClip(c echo.Context) error {
	createClipDto := new(CreateClipDTO)
	if err := c.Bind(createClipDto); err != nil {
		c.Logger().Errorf("Invalid input: could not bind request: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
	}

	if err := validateCreateClipDto(createClipDto); err != nil {
		c.Logger().Errorf("Invalid DTO: %s", err.Error())
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	job := jobs.NewJob()
	go videoprocessing.ProcessClip(job.ID, createClipDto.Url, createClipDto.From, createClipDto.To, createClipDto.Format)

	return c.String(http.StatusCreated, job.ID)
}

func GetClip(c echo.Context) error {
	jobID := c.QueryParam("jobId")
	job, exists := jobs.GetJobById(jobID)
	if !exists || job.Status != jobs.StatusCompleted {
		c.Logger().Errorf("Job does not exist or is not completed. JobId:%s", jobID)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Job does not exist"})
	}

	return c.Attachment(job.FilePath, filepath.Base(job.FilePath))
}
