package api

import (
	"net/http"
	"path/filepath"
	"strings"
	"ytclipper-go/config"
	"ytclipper-go/jobs"
	"ytclipper-go/videoprocessing"

	"github.com/labstack/echo/v4"
)

// errServerBusy is returned with 429 when the job queue is full.
const errServerBusy = "The server is busy, try again in a minute."

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

	job, ok := jobs.TryNewJob(config.CONFIG.MaxQueuedJobs)
	if !ok {
		c.Logger().Warnf("Job queue full (%d jobs), rejecting clip request", config.CONFIG.MaxQueuedJobs)
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": errServerBusy})
	}
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
	c.Response().Header().Set(echo.HeaderContentDisposition, attachmentDisposition(name))
	return c.File(job.FilePath)
}

// attachmentDisposition builds a Content-Disposition header for a download
// name that may contain any Unicode: an ASCII fallback in filename for old
// clients, and the real name in filename* (RFC 6266, encoded per RFC 5987).
func attachmentDisposition(name string) string {
	return `attachment; filename="` + asciiFileName(name) + `"; filename*=UTF-8''` + encodeRFC5987(name)
}

// asciiFileName replaces everything outside printable ASCII, the characters
// that would end or escape the quoted string, and "%" (which some clients
// percent-decode) with "_".
func asciiFileName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' || r == '%' {
			b.WriteByte('_')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// encodeRFC5987 percent-encodes every UTF-8 byte that is not an attr-char
// (RFC 5987 section 3.2.1).
func encodeRFC5987(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if isAttrChar(ch) {
			b.WriteByte(ch)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[ch>>4])
		b.WriteByte(hex[ch&0x0f])
	}
	return b.String()
}

func isAttrChar(ch byte) bool {
	switch {
	case 'a' <= ch && ch <= 'z', 'A' <= ch && ch <= 'Z', '0' <= ch && ch <= '9':
		return true
	}
	return strings.IndexByte("!#$&+-.^_`|~", ch) >= 0
}
