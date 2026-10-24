package api

import (
	"encoding/json"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MorrisMorrison/ytclipper-go/internal/config"
	"github.com/MorrisMorrison/ytclipper-go/internal/jobs"

	"github.com/labstack/echo/v4"
)

func serve(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	e.POST("/api/v1/clip", CreateClip)
	e.GET("/api/v1/clip", GetClip)
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

func TestCreateClipRejectsWhenQueueIsFull(t *testing.T) {
	original := config.CONFIG.MaxQueuedJobs
	config.CONFIG.MaxQueuedJobs = jobs.ActiveCount() // already full
	t.Cleanup(func() { config.CONFIG.MaxQueuedJobs = original })

	rec := serve(t, http.MethodPost, "/api/v1/clip", `{"url":"https://youtu.be/dQw4w9WgXcQ","from":"00:00:01","to":"00:00:05","preset":"video"}`)
	if rec.Code != http.StatusTooManyRequests || decode(t, rec)["error"] != "The server is busy, try again in a minute." {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}

	// Invalid requests are still answered with their validation error.
	rec = serve(t, http.MethodPost, "/api/v1/clip", `{"url":"https://youtu.be/dQw4w9WgXcQ","from":"00:00:01","to":"00:00:05"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
}

func TestCreateClipRejectsTooLongClip(t *testing.T) {
	original := config.CONFIG.MaxClipLengthInMinutes
	config.CONFIG.MaxClipLengthInMinutes = 5
	t.Cleanup(func() { config.CONFIG.MaxClipLengthInMinutes = original })

	rec := serve(t, http.MethodPost, "/api/v1/clip", `{"url":"https://youtu.be/dQw4w9WgXcQ","from":"00:00:00","to":"00:05:01","preset":"audio"}`)
	if rec.Code != http.StatusBadRequest || decode(t, rec)["error"] != "Clip is too long (max 5 minutes)." {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
}

func TestGetClipContentDisposition(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clip.mp4")
	if err := os.WriteFile(path, []byte("clip"), 0o644); err != nil {
		t.Fatal(err)
	}
	job := jobs.NewJob()
	jobs.CompleteJob(job.ID, path, "日本語 🎵 50% [0-00-10 - 0-00-20].mp4")

	rec := serve(t, http.MethodGet, "/api/v1/clip?jobId="+job.ID, "")
	if rec.Code != http.StatusOK || rec.Body.String() != "clip" {
		t.Fatalf("got %d %q", rec.Code, rec.Body)
	}
	want := `attachment; filename="___ _ 50_ [0-00-10 - 0-00-20].mp4"; filename*=UTF-8''%E6%97%A5%E6%9C%AC%E8%AA%9E%20%F0%9F%8E%B5%2050%25%20%5B0-00-10%20-%200-00-20%5D.mp4`
	if got := rec.Header().Get(echo.HeaderContentDisposition); got != want {
		t.Errorf("Content-Disposition =\n%s\nwant\n%s", got, want)
	}

	_, params, err := mime.ParseMediaType(rec.Header().Get(echo.HeaderContentDisposition))
	if err != nil || params["filename"] != "日本語 🎵 50% [0-00-10 - 0-00-20].mp4" {
		t.Errorf("a standard parser reads %q (%v)", params["filename"], err)
	}
}

func TestEncodeRFC5987(t *testing.T) {
	if got := encodeRFC5987("a-Z_0.9!#$&+^`|~"); got != "a-Z_0.9!#$&+^`|~" {
		t.Errorf("attr-chars must stay: %q", got)
	}
	if got := encodeRFC5987(`"'();,/:<=>?@[\]{} *%`); got != "%22%27%28%29%3B%2C%2F%3A%3C%3D%3E%3F%40%5B%5C%5D%7B%7D%20%2A%25" {
		t.Errorf("everything else must be encoded: %q", got)
	}
	if got := asciiFileName("é\"\\%x"); got != "____x" {
		t.Errorf("asciiFileName = %q", got)
	}
}

func TestGetJobStatusLegacy(t *testing.T) {
	if rec := serve(t, http.MethodGet, "/api/v1/jobs/status?jobId=nope", ""); rec.Code != http.StatusNotFound {
		t.Errorf("unknown job: got %d", rec.Code)
	}

	queued := jobs.NewJob()
	if rec := serve(t, http.MethodGet, "/api/v1/jobs/status?jobId="+queued.ID, ""); rec.Code != http.StatusCreated {
		t.Errorf("queued job: got %d; want 201", rec.Code)
	}

	done := jobs.NewJob()
	jobs.CompleteJob(done.ID, "x.mp4", "x.mp4")
	if rec := serve(t, http.MethodGet, "/api/v1/jobs/status?jobId="+done.ID, ""); rec.Code != http.StatusOK {
		t.Errorf("completed job: got %d; want 200", rec.Code)
	}

	failed := jobs.NewJob()
	jobs.FailJob(failed.ID, "boom")
	rec := serve(t, http.MethodGet, "/api/v1/jobs/status?jobId="+failed.ID, "")
	if rec.Code != http.StatusInternalServerError || decode(t, rec)["error"] != "boom" {
		t.Errorf("failed job: got %d %s; want 500 with the error", rec.Code, rec.Body)
	}
}

func TestGetVideoInfoRequiresURL(t *testing.T) {
	rec := serve(t, http.MethodGet, "/api/v1/video/info", "")
	if rec.Code != http.StatusBadRequest || decode(t, rec)["error"] != "URL is required" {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
}

func TestGetClipUnknownOrUnfinishedJob(t *testing.T) {
	if rec := serve(t, http.MethodGet, "/api/v1/clip?jobId=nope", ""); rec.Code != http.StatusNotFound {
		t.Errorf("unknown job: got %d; want 404", rec.Code)
	}
	queued := jobs.NewJob()
	if rec := serve(t, http.MethodGet, "/api/v1/clip?jobId="+queued.ID, ""); rec.Code != http.StatusNotFound {
		t.Errorf("unfinished job: got %d; want 404", rec.Code)
	}
}

func TestCreateClipRejectsBadJSON(t *testing.T) {
	rec := serve(t, http.MethodPost, "/api/v1/clip", `{"url":`)
	if rec.Code != http.StatusBadRequest || decode(t, rec)["error"] != "Invalid input" {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
}
