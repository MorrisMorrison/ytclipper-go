package api

import (
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
            c.Logger().Errorf("Invalid input: Could not read json body")
            return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
        }

        c.Logger().Errorf("Invalid input: Could not bind to DTO. Body:/n%s", rawJSON.String())
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
    }

    if err := validateCreateClipDto(createClipDto); err != nil {
        c.Logger().Errorf("Invalid DTO: %s", err.Error())
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
        c.Logger().Errorf("Job does not exist. JobId:%s", jobID)
        return c.JSON(http.StatusNotFound, map[string]string{"error": "Job does not exist"})
    }

    return c.File(job.FilePath)
}

