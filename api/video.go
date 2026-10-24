package api

import (
	"fmt"
	"net/http"
	"ytclipper-go/videoprocessing"

	"github.com/labstack/echo/v4"
)

// videoInfo loads the (cached) info for the video in the youtubeUrl query
// parameter. When that fails it writes the error response and returns
// ok=false along with the result of writing it.
func videoInfo(c echo.Context, errorStatus int) (info *videoprocessing.VideoInfo, ok bool, err error) {
	if c.QueryParam("youtubeUrl") == "" {
		return nil, false, c.JSON(http.StatusBadRequest, map[string]string{"error": "URL is required"})
	}

	videoID, ok := extractVideoID(c.QueryParam("youtubeUrl"))
	if !ok {
		c.Logger().Errorf("Invalid Youtube URL: %s", c.QueryParam("youtubeUrl"))
		return nil, false, c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid YouTube URL"})
	}

	info, err = videoprocessing.GetVideoInfo(videoID)
	if err != nil {
		return nil, false, c.JSON(errorStatus, map[string]string{"error": err.Error()})
	}
	return info, true, nil
}

// GetVideoInfo returns title, duration, thumbnail and formats of a video.
func GetVideoInfo(c echo.Context) error {
	info, ok, err := videoInfo(c, http.StatusBadGateway)
	if !ok {
		return err
	}
	return c.JSON(http.StatusOK, info)
}

// GetVideoDuration returns the video length in seconds (used by the old page).
func GetVideoDuration(c echo.Context) error {
	info, ok, err := videoInfo(c, http.StatusInternalServerError)
	if !ok {
		return err
	}
	return c.JSON(http.StatusOK, info.Duration)
}

// legacyFormat is the format shape the old page's dropdown expects.
type legacyFormat struct {
	ID         string `json:"id"`
	Extension  string `json:"extension"`
	Label      string `json:"label"`
	Codec      string `json:"codec"`
	Bitrate    string `json:"bitrate"`
	FormatType string `json:"formatType"`
	Additional string `json:"additional"`
}

func toLegacyFormat(f videoprocessing.Format) legacyFormat {
	lf := legacyFormat{
		ID:         f.ID,
		Extension:  f.Ext,
		Codec:      f.VCodec,
		Bitrate:    "N/A",
		Additional: f.Note,
	}

	switch f.Kind {
	case videoprocessing.FormatKindAudio:
		lf.FormatType, lf.Label, lf.Codec = "audio only", "audio only", f.ACodec
	case videoprocessing.FormatKindVideo:
		lf.FormatType = "video only"
	default:
		lf.FormatType = "audio and video"
	}
	if lf.Label == "" {
		lf.Label = f.Note
		if f.Width > 0 && f.Height > 0 {
			lf.Label = fmt.Sprintf("%dx%d", f.Width, f.Height)
		}
	}
	if f.TBR > 0 {
		lf.Bitrate = fmt.Sprintf("%.0fk", f.TBR)
	}
	return lf
}

// GetAvailableFormats returns the formats in the old page's shape.
func GetAvailableFormats(c echo.Context) error {
	info, ok, err := videoInfo(c, http.StatusInternalServerError)
	if !ok {
		return err
	}

	formats := make([]legacyFormat, 0, len(info.Formats))
	for _, f := range info.Formats {
		formats = append(formats, toLegacyFormat(f))
	}
	return c.JSON(http.StatusOK, formats)
}
