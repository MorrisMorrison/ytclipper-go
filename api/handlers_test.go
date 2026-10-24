package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"ytclipper-go/jobs"
	"ytclipper-go/videoprocessing"

	"github.com/labstack/echo/v4"
)

func serve(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	e.POST("/api/v1/clip", CreateClip)
	e.GET("/api/v1/jobs/status", GetJobStatus)
	e.GET("/api/v1/jobs/:id", GetJob)
	e.GET("/api/v1/video/info", GetVideoInfo)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", rec.Body.String(), err)
	}
	return m
}

func TestGetJob(t *testing.T) {
	if rec := serve(t, http.MethodGet, "/api/v1/jobs/nope", ""); rec.Code != http.StatusNotFound || decode(t, rec)["error"] != "Job not found" {
		t.Errorf("unknown job: %d %s", rec.Code, rec.Body)
	}

	queued := jobs.NewJob()
	rec := serve(t, http.MethodGet, "/api/v1/jobs/"+queued.ID, "")
	got := decode(t, rec)
	if rec.Code != http.StatusOK || got["status"] != "queued" || got["error"] != "" {
		t.Errorf("queued job: %d %v", rec.Code, got)
	}
	if _, ok := got["fileName"]; ok {
		t.Errorf("fileName must be absent until completed: %v", got)
	}

	done := jobs.NewJob()
	jobs.CompleteJob(done.ID, "/clips/x.mp4", "Title [0-00-01 - 0-00-02].mp4")
	got = decode(t, serve(t, http.MethodGet, "/api/v1/jobs/"+done.ID, ""))
	if got["status"] != "completed" || got["fileName"] != "Title [0-00-01 - 0-00-02].mp4" || got["downloadUrl"] != "/api/v1/clip?jobId="+done.ID {
		t.Errorf("completed job: %v", got)
	}

	failed := jobs.NewJob()
	jobs.FailJob(failed.ID, "Video unavailable")
	got = decode(t, serve(t, http.MethodGet, "/api/v1/jobs/"+failed.ID, ""))
	if got["status"] != "error" || got["error"] != "Video unavailable" {
		t.Errorf("failed job: %v", got)
	}

	// The old status route still wins over /jobs/:id.
	if rec := serve(t, http.MethodGet, "/api/v1/jobs/status?jobId="+done.ID, ""); rec.Code != http.StatusOK {
		t.Errorf("old status endpoint: %d", rec.Code)
	}
}

func TestCreateClipValidation(t *testing.T) {
	rec := serve(t, http.MethodPost, "/api/v1/clip", `{"url":"https://youtu.be/dQw4w9WgXcQ","from":"00:00:01","to":"00:00:05","preset":"video","format":"18"}`)
	if rec.Code != http.StatusBadRequest || decode(t, rec)["error"] != "Specify either a preset or a format, not both." {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
}

func TestGetVideoInfoRejectsInvalidURL(t *testing.T) {
	rec := serve(t, http.MethodGet, "/api/v1/video/info?youtubeUrl=https://vimeo.com/1", "")
	if rec.Code != http.StatusBadRequest || decode(t, rec)["error"] != "Invalid YouTube URL" {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
}

func TestToLegacyFormat(t *testing.T) {
	tests := []struct {
		in   videoprocessing.Format
		want legacyFormat
	}{
		{
			videoprocessing.Format{ID: "140", Ext: "m4a", Kind: "audio", ACodec: "mp4a.40.2", TBR: 129.5, Note: "medium"},
			legacyFormat{ID: "140", Extension: "m4a", Label: "audio only", Codec: "mp4a.40.2", Bitrate: "130k", FormatType: "audio only", Additional: "medium"},
		},
		{
			videoprocessing.Format{ID: "136", Ext: "mp4", Kind: "video", Width: 1280, Height: 720, VCodec: "avc1.4d401f", Note: "720p"},
			legacyFormat{ID: "136", Extension: "mp4", Label: "1280x720", Codec: "avc1.4d401f", Bitrate: "N/A", FormatType: "video only", Additional: "720p"},
		},
		{
			videoprocessing.Format{ID: "18", Ext: "mp4", Kind: "both", Width: 640, Height: 360, VCodec: "avc1.42001E", ACodec: "mp4a.40.2", TBR: 492.5, Note: "360p"},
			legacyFormat{ID: "18", Extension: "mp4", Label: "640x360", Codec: "avc1.42001E", Bitrate: "492k", FormatType: "audio and video", Additional: "360p"},
		},
	}
	for _, tt := range tests {
		if got := toLegacyFormat(tt.in); got != tt.want {
			t.Errorf("toLegacyFormat(%s) = %+v; want %+v", tt.in.ID, got, tt.want)
		}
	}
}
