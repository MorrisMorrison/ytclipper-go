package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MorrisMorrison/ytclipper-go/internal/config"
)

func get(t *testing.T, devDir string, path string) *httptest.ResponseRecorder {
	t.Helper()
	original := *config.CONFIG
	t.Cleanup(func() { *config.CONFIG = original })
	config.CONFIG.BasicAuthConfig = config.BasicAuthConfig{}
	config.CONFIG.TrustedProxies = nil
	config.CONFIG.DevAssetsDir = devDir

	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

// The page and static files come from the binary, whatever the working
// directory (tests run in internal/server, not the repository root).
func TestServesEmbeddedAssets(t *testing.T) {
	rec := get(t, "", "/")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "/static/scripts/main.js") {
		t.Fatalf("/: got %d, body %.200q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("/: Content-Type %q", ct)
	}

	for path, contentType := range map[string]string{
		"/static/css/styles.css":    "text/css",
		"/static/scripts/main.js":   "javascript",
		"/static/icons/favicon.ico": "",
	} {
		rec := get(t, "", path)
		if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
			t.Errorf("%s: got %d, %d bytes", path, rec.Code, rec.Body.Len())
		}
		if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, contentType) {
			t.Errorf("%s: Content-Type %q", path, ct)
		}
	}

	if rec := get(t, "", "/static/nope.js"); rec.Code != http.StatusNotFound {
		t.Errorf("missing static file: got %d; want 404", rec.Code)
	}
	if rec := get(t, "", "/static/../embed.go"); rec.Code == http.StatusOK {
		t.Errorf("path traversal: got %d", rec.Code)
	}
}

// YTCLIPPER_DEV_ASSETS_DIR serves from disk and picks up edits without a
// restart.
func TestServesDevAssetsFromDisk(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("templates/index.html", "<p>first</p>")
	write("static/app.js", "console.log(1)")

	original := *config.CONFIG
	t.Cleanup(func() { *config.CONFIG = original })
	config.CONFIG.BasicAuthConfig = config.BasicAuthConfig{}
	config.CONFIG.DevAssetsDir = dir
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	body := func(path string) string {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: got %d", path, rec.Code)
		}
		return rec.Body.String()
	}

	if got := body("/"); got != "<p>first</p>" {
		t.Errorf("/: got %q", got)
	}
	if got := body("/static/app.js"); got != "console.log(1)" {
		t.Errorf("static: got %q", got)
	}
	write("templates/index.html", "<p>second</p>")
	if got := body("/"); got != "<p>second</p>" {
		t.Errorf("after an edit: got %q", got)
	}
}

// A YouTube link with the host swapped for ours opens the page; anything that
// isn't a video ID keeps returning 404.
func TestDomainSwapRoutes(t *testing.T) {
	tests := []struct {
		path string
		want int
	}{
		{"/watch?v=dQw4w9WgXcQ", http.StatusOK},
		{"/watch?v=dQw4w9WgXcQ&list=PL123&t=42s", http.StatusOK},
		{"/shorts/dQw4w9WgXcQ", http.StatusOK},
		{"/live/dQw4w9WgXcQ?si=abc", http.StatusOK},
		{"/dQw4w9WgXcQ", http.StatusOK},
		{"/dQw4w9WgXcQ?si=abc&t=90", http.StatusOK},
		{"/watch", http.StatusNotFound},
		{"/watch?v=short", http.StatusNotFound},
		{"/shorts/not-an-id", http.StatusNotFound},
		{"/favicon.ico", http.StatusNotFound},
		{"/robots.txt", http.StatusNotFound},
		{"/nope", http.StatusNotFound},
		{"/dQw4w9WgXcQ/extra", http.StatusNotFound},
	}
	for _, tt := range tests {
		rec := get(t, "", tt.path)
		if rec.Code != tt.want {
			t.Errorf("GET %s = %d; want %d", tt.path, rec.Code, tt.want)
			continue
		}
		if tt.want == http.StatusOK && !strings.Contains(rec.Body.String(), "/static/scripts/main.js") {
			t.Errorf("GET %s did not serve the page", tt.path)
		}
	}

	// Static routes win over the /:id catch-all.
	if rec := get(t, "", "/health"); rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), "Server is running") {
		t.Errorf("/health = %d %q", rec.Code, rec.Body.String())
	}
}
