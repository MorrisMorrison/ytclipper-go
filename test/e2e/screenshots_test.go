//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// Big Buck Bunny (CC BY 3.0, Blender Foundation) on YouTube. Its license
// allows showing it in the README.
const (
	screenshotVideoID  = "aqz-KE-bpKQ"
	screenshotTitle    = "Big Buck Bunny"
	screenshotUploader = "Blender"
	screenshotDuration = "635" // 10:35, as the YouTube player reports it
)

// TestScreenshots renders the README images. It runs only with
// E2E_SCREENSHOTS=<dir> (make screenshots) and, unlike the rest of the suite,
// lets the browser reach YouTube so the real player shows. The fake yt-dlp
// still answers the server, with Big Buck Bunny's title and length.
func TestScreenshots(t *testing.T) {
	outDir := os.Getenv("E2E_SCREENSHOTS")
	if outDir == "" {
		t.Skip("set E2E_SCREENSHOTS=<dir> (or run make screenshots) to render the README images")
	}
	outDir, err := filepath.Abs(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(outDir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}

	s := ownServer(t, "screenshots", map[string]string{
		"FAKE_YTDLP_TITLE":    screenshotTitle,
		"FAKE_YTDLP_UPLOADER": screenshotUploader,
		"FAKE_YTDLP_DURATION": screenshotDuration,
	})
	b, err := startBrowser(false)
	if err != nil {
		t.Fatal(err)
	}

	for _, theme := range []string{"light", "dark"} {
		t.Run(theme, func(t *testing.T) {
			p := newPage(t, withServer(s.URL), withBrowser(b), withColorScheme(theme), withViewport(1280, 900))
			// Type the range before the player is ready: typed times seek a
			// ready player, which would replace the thumbnail with a frame
			// that headless Chrome may not render.
			p.enterURL(watchURL(screenshotVideoID))
			if state := eval[string](p, `document.querySelector(".player-frame").dataset.player`); state == "ready" {
				t.Log("the player was ready before the range was typed; the image may show a video frame")
			}
			p.setRange("1:00", "2:30")
			poll[bool](p, "the video length did not arrive", `document.getElementById("trimmer").dataset.ready === "true"`, 15*time.Second)

			// The player is optional; a run without YouTube still renders,
			// with the fallback.
			if _, err := chromedp.Run(p.ctx, chromedp.Poll[bool](
				`document.querySelector(".player-frame").dataset.player === "ready"`,
				chromedp.WithPollingInterval(200*time.Millisecond), chromedp.WithPollingTimeout(20*time.Second))); err != nil {
				t.Logf("the YouTube player did not load (%v); rendering without it", err)
			}

			p.clip()
			// Let the thumbnail load and the smooth scroll to the job panel end.
			p.do("waiting", chromedp.Sleep(2*time.Second), chromedp.Evaluate[chromedp.Void](`window.scrollTo(0, 0); undefined`))

			png, err := chromedp.Run(p.ctx, chromedp.FullScreenshot(100))
			if err != nil {
				t.Fatalf("screenshot: %v", err)
			}
			path := filepath.Join(outDir, "docs", "screenshot-"+theme+".png")
			if err := os.WriteFile(path, png, 0o644); err != nil {
				t.Fatal(err)
			}
			t.Logf("wrote %s", path)
		})
	}
}
