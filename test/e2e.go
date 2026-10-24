// End-to-end tests for ytclipper-go.
//
// Drives a headless Chrome against a running server (CI puts the yt-dlp stub
// test/ci/yt-dlp first on PATH, so the whole page → API → job → download flow
// runs offline). The YouTube player iframe is never required: it usually does
// not load in CI, so every test types the times and works without it.
//
// Environment variables:
//   - E2E_BASE_URL: server to test (default http://localhost:8080)
//   - E2E_DOWNLOAD_TIMEOUT: seconds to wait for a clip job (default 25)
//   - E2E_EXPECT_FAILURE: accept a job that ends in an error, for runs
//     against real YouTube where downloads may be blocked (default false)
//
// Usage:
//
//	go run test/e2e.go
//	E2E_DOWNLOAD_TIMEOUT=60 E2E_EXPECT_FAILURE=true go run test/e2e.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)

const (
	defaultBaseURL = "http://localhost:8080"

	urlInput       = chromedp.CSS(`#url`)
	urlError       = chromedp.CSS(`#urlError`)
	editor         = chromedp.CSS(`#editor`)
	fromInput      = chromedp.CSS(`#from`)
	toInput        = chromedp.CSS(`#to`)
	timeError      = chromedp.CSS(`#timeError`)
	presetAudio    = chromedp.CSS(`#presetAudio`)
	advancedToggle = chromedp.CSS(`#advanced summary`)
	videoFormat137 = chromedp.CSS(`#videoFormats input[value="137"]`)
	audioFormat140 = chromedp.CSS(`#audioFormats input[value="140"]`)
	formatSummary  = chromedp.CSS(`#formatSummary`)
	clipButton     = chromedp.CSS(`#clipButton`)
	downloadButton = chromedp.CSS(`#downloadButton`)
	jobDetail      = chromedp.CSS(`#jobDetail`)

	validYouTubeURL   = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	invalidYouTubeURL = "invalid-url"

	defaultDownloadTimeoutSeconds = 25
)

func baseURL() string {
	if v := os.Getenv("E2E_BASE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return defaultBaseURL
}

func downloadTimeout() time.Duration {
	if v, err := strconv.Atoi(os.Getenv("E2E_DOWNLOAD_TIMEOUT")); err == nil && v > 0 {
		return time.Duration(v) * time.Second
	}
	return defaultDownloadTimeoutSeconds * time.Second
}

func expectDownloadFailure() bool {
	v, err := strconv.ParseBool(os.Getenv("E2E_EXPECT_FAILURE"))
	return err == nil && v
}

type Test struct {
	Name string
	Run  func(ctx context.Context) error
}

func main() {
	tests := []Test{
		{Name: "Simple mode: video clip", Run: testVideoClip},
		{Name: "Simple mode: audio preset", Run: testAudioClip},
		{Name: "Advanced mode: chosen formats", Run: testAdvancedClip},
		{Name: "Invalid URL shows an error", Run: testInvalidURL},
		{Name: "End before start is rejected", Run: testEndBeforeStart},
	}

	failed := 0
	for _, test := range tests {
		// Each test gets its own browser, torn down before the next one starts.
		if err := runTest(test); err != nil {
			failed++
		}
	}
	if failed > 0 {
		log.Printf("%d test(s) failed", failed)
		os.Exit(1)
	}
	log.Println("All tests passed")
}

func runTest(test Test) error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.WindowSize(1280, 1000),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		if strings.Contains(msg, "could not unmarshal event") {
			return
		}
		log.Printf("Chrome: %s", msg)
	}))
	defer cancel()

	testCtx, cancelTest := context.WithTimeout(ctx, 2*time.Minute)
	defer cancelTest()

	// The page starts the download on its own when a clip is ready; don't
	// write files from the test run.
	if err := chromedp.Do(testCtx); err != nil {
		log.Printf("Test '%s' failed: starting Chrome: %v", test.Name, err)
		return err
	}
	if _, err := chromedp.CallBrowser(testCtx, browser.SetDownloadBehavior, browser.SetDownloadBehaviorParams{
		Behavior: browser.SetDownloadBehaviorBehaviorDeny,
	}); err != nil {
		log.Printf("could not disable downloads (continuing): %v", err)
	}

	log.Printf("Running test: %s", test.Name)
	start := time.Now()
	err := test.Run(testCtx)
	if err != nil {
		log.Printf("Test '%s' failed after %v: %v", test.Name, time.Since(start), err)
		return err
	}
	log.Printf("Test '%s' passed in %v", test.Name, time.Since(start))
	return nil
}

// ===== helpers =====

// openVideo loads the page and pastes a link; the editor appears without any
// server round trip.
func openVideo(ctx context.Context) error {
	if err := chromedp.Do(ctx,
		chromedp.Navigate(baseURL()),
		chromedp.WaitVisible(urlInput),
		chromedp.SendKeys(urlInput, validYouTubeURL),
		chromedp.WaitVisible(editor),
	); err != nil {
		return fmt.Errorf("editor did not appear after entering a URL (is the server running on %s?): %w", baseURL(), err)
	}
	// Give the background video info call a moment to set the duration, so it
	// doesn't land while we type. Not required: the page works without it.
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := chromedp.Run(waitCtx, chromedp.Poll[bool](`document.getElementById("trimmer").dataset.ready === "true"`,
		chromedp.WithPollingInterval(200*time.Millisecond), chromedp.WithPollingTimeout(10*time.Second))); err != nil {
		log.Println("video duration not known yet; continuing with typed times")
	}
	return nil
}

// typeTime replaces a time field's value with real keystrokes and blurs it,
// which commits it the way a user leaving the field does.
func typeTime(sel chromedp.CSS, value string) chromedp.Action[chromedp.Void] {
	return chromedp.Steps(
		// SetValue refuses an empty string, so clear the field from script.
		chromedp.Evaluate[chromedp.Void](fmt.Sprintf(`document.querySelector(%q).value = ""; undefined`, string(sel))),
		chromedp.SendKeys(sel, value),
		chromedp.Blur(sel),
	)
}

func setRange(ctx context.Context, from, to string) error {
	if err := chromedp.Do(ctx, typeTime(fromInput, from), typeTime(toInput, to)); err != nil {
		return fmt.Errorf("could not type the time range: %w", err)
	}
	return nil
}

func isHidden(ctx context.Context, id string) (bool, error) {
	return chromedp.Run(ctx, chromedp.Evaluate[bool](fmt.Sprintf(`document.getElementById(%q).hidden`, id)))
}

// clipAndWait clicks "Create clip" and waits for the job to finish. It
// returns the file name the server reported.
func clipAndWait(ctx context.Context) (string, error) {
	if err := chromedp.Do(ctx, chromedp.Click(clipButton)); err != nil {
		return "", fmt.Errorf("could not click Create clip: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, downloadTimeout())
	defer cancel()
	state, err := chromedp.Run(waitCtx, chromedp.Poll[string](
		`(s => s === "completed" || s === "error" ? s : false)(document.getElementById("job").dataset.state)`,
		chromedp.WithPollingInterval(250*time.Millisecond), chromedp.WithPollingTimeout(downloadTimeout())))
	if err != nil {
		return "", fmt.Errorf("job did not finish within %v: %w", downloadTimeout(), err)
	}
	detail, _ := chromedp.Run(ctx, chromedp.Text(jobDetail))

	if state == "error" {
		if expectDownloadFailure() {
			log.Printf("job failed as allowed by E2E_EXPECT_FAILURE: %s", detail)
			return "", nil
		}
		return "", fmt.Errorf("job failed: %s", detail)
	}

	if err := chromedp.Do(ctx, chromedp.WaitVisible(downloadButton)); err != nil {
		return "", fmt.Errorf("download button not shown: %w", err)
	}
	href, err := chromedp.Run(ctx, chromedp.AttributeValue(downloadButton, "href"))
	if err != nil || !strings.Contains(href.Value, "/api/v1/clip?jobId=") {
		return "", fmt.Errorf("unexpected download link %q (%v)", href.Value, err)
	}
	log.Printf("clip ready: %s (%s)", detail, href.Value)
	return detail, nil
}

func expectExtension(fileName, ext string) error {
	if fileName == "" && expectDownloadFailure() {
		return nil
	}
	if !strings.HasSuffix(strings.ToLower(fileName), ext) {
		return fmt.Errorf("expected a %s file, got %q", ext, fileName)
	}
	return nil
}

// ===== tests =====

func testVideoClip(ctx context.Context) error {
	if err := openVideo(ctx); err != nil {
		return err
	}
	if err := setRange(ctx, "10", "20"); err != nil {
		return err
	}
	fileName, err := clipAndWait(ctx)
	if err != nil {
		return err
	}
	return expectExtension(fileName, ".mp4")
}

func testAudioClip(ctx context.Context) error {
	if err := openVideo(ctx); err != nil {
		return err
	}
	if err := setRange(ctx, "0:05", "0:15"); err != nil {
		return err
	}
	if err := chromedp.Do(ctx, chromedp.Click(presetAudio)); err != nil {
		return fmt.Errorf("could not choose the audio preset: %w", err)
	}
	fileName, err := clipAndWait(ctx)
	if err != nil {
		return err
	}
	return expectExtension(fileName, ".mp3")
}

func testAdvancedClip(ctx context.Context) error {
	if err := openVideo(ctx); err != nil {
		return err
	}
	if err := setRange(ctx, "10", "20"); err != nil {
		return err
	}
	// The format list is only fetched when the panel is opened.
	if err := chromedp.Do(ctx,
		chromedp.Click(advancedToggle),
		chromedp.WaitReady(videoFormat137),
		chromedp.Click(videoFormat137),
		chromedp.Click(audioFormat140),
	); err != nil {
		return fmt.Errorf("advanced formats did not load: %w", err)
	}
	summary, err := chromedp.Run(ctx, chromedp.Text(formatSummary))
	if err != nil {
		return err
	}
	if summary != "137+140" {
		return fmt.Errorf("expected format 137+140, got %q", summary)
	}
	_, err = clipAndWait(ctx)
	return err
}

func testInvalidURL(ctx context.Context) error {
	if err := chromedp.Do(ctx,
		chromedp.Navigate(baseURL()),
		chromedp.WaitVisible(urlInput),
		chromedp.SendKeys(urlInput, invalidYouTubeURL),
		chromedp.Blur(urlInput),
		chromedp.WaitVisible(urlError),
	); err != nil {
		return fmt.Errorf("no error shown for an invalid URL: %w", err)
	}
	hidden, err := isHidden(ctx, "editor")
	if err != nil {
		return err
	}
	if !hidden {
		return fmt.Errorf("editor is shown for an invalid URL")
	}
	return nil
}

func testEndBeforeStart(ctx context.Context) error {
	if err := openVideo(ctx); err != nil {
		return err
	}
	if err := setRange(ctx, "20", "10"); err != nil {
		return err
	}
	if err := chromedp.Do(ctx, chromedp.Click(clipButton), chromedp.WaitVisible(timeError)); err != nil {
		return fmt.Errorf("no error shown for end before start: %w", err)
	}
	hidden, err := isHidden(ctx, "job")
	if err != nil {
		return err
	}
	if !hidden {
		return fmt.Errorf("a job was started for an invalid range")
	}
	return nil
}
