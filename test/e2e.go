// End-to-end tests for ytclipper-go.
//
// Drives a headless Chrome against a running server (CI puts the yt-dlp stub
// test/ci/yt-dlp first on PATH, so the whole page → API → job → download flow
// runs offline). The browser can't resolve YouTube hosts, so the player never
// loads: the "player unavailable" path is what's tested, and the video length
// always comes from the stub's /video/info (30 s). Every test types the times.
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
	"github.com/chromedp/chromedp/kb"
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
	videoFormat136 = chromedp.CSS(`#videoFormats input[value="136"]`)
	videoFormat18  = chromedp.CSS(`#videoFormats input[value="18"]`)
	audioFormat251 = chromedp.CSS(`#audioFormats input[value="251"]`)
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
		// No YouTube player or thumbnails: keeps runs offline and deterministic.
		chromedp.Flag("host-resolver-rules", "MAP youtube.com ~NOTFOUND, MAP *.youtube.com ~NOTFOUND, "+
			"MAP youtube-nocookie.com ~NOTFOUND, MAP *.youtube-nocookie.com ~NOTFOUND, MAP *.ytimg.com ~NOTFOUND"),
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
		recordClipRequests,
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

// recordClipRequests wraps fetch so tests can check what the page POSTed to
// /api/v1/clip (window.__clipPosts).
var recordClipRequests = chromedp.Evaluate[chromedp.Void](`(() => {
	window.__clipPosts = [];
	const original = window.fetch;
	window.fetch = (url, options = {}) => {
		if (String(url).endsWith("/api/v1/clip") && options.method === "POST") window.__clipPosts.push(JSON.parse(options.body));
		return original(url, options);
	};
})()`)

type clipPayload struct {
	Preset string `json:"preset"`
	Format string `json:"format"`
	From   string `json:"from"`
	To     string `json:"to"`
}

func clipPosts(ctx context.Context) ([]clipPayload, error) {
	return chromedp.Run(ctx, chromedp.Evaluate[[]clipPayload](`window.__clipPosts || []`))
}

// lastClipPost returns the last /api/v1/clip payload the page sent.
func lastClipPost(ctx context.Context) (clipPayload, error) {
	posts, err := clipPosts(ctx)
	if err != nil {
		return clipPayload{}, err
	}
	if len(posts) == 0 {
		return clipPayload{}, fmt.Errorf("the page sent no clip request")
	}
	return posts[len(posts)-1], nil
}

// enterTime focuses a time field, then replaces its value with real
// keystrokes. Focusing first matters: the page doesn't rewrite a focused field
// when the video length arrives.
func enterTime(sel chromedp.CSS, value string) chromedp.Action[chromedp.Void] {
	return chromedp.Steps(
		chromedp.Focus(sel),
		// SetValue refuses an empty string, so clear the field from script.
		chromedp.Evaluate[chromedp.Void](fmt.Sprintf(`document.querySelector(%q).value = ""; undefined`, string(sel))),
		chromedp.SendKeys(sel, value),
	)
}

// typeTime enters a time and leaves the field, which commits it the way a
// user does.
func typeTime(sel chromedp.CSS, value string) chromedp.Action[chromedp.Void] {
	return chromedp.Steps(enterTime(sel, value), chromedp.Blur(sel))
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
	// YouTube is unreachable here, so the page must say so and still know the
	// length from /video/info.
	if _, err := chromedp.Run(ctx, chromedp.Poll[bool](`document.querySelector(".player-frame").dataset.player === "unavailable"`,
		chromedp.WithPollingTimeout(20*time.Second))); err != nil {
		return fmt.Errorf("player-unavailable state not shown: %w", err)
	}
	length, err := chromedp.Run(ctx, chromedp.Text(chromedp.CSS(`#trimDuration`)))
	if err != nil || length != "0:30" {
		return fmt.Errorf("expected the stub's length 0:30 on the slider, got %q (%v)", length, err)
	}
	if err := setRange(ctx, "10", "20"); err != nil {
		return err
	}
	fileName, err := clipAndWait(ctx)
	if err != nil {
		return err
	}
	post, err := lastClipPost(ctx)
	if err != nil {
		return err
	}
	if post.Preset != "video" || post.Format != "" || post.From != "00:00:10" || post.To != "00:00:20" {
		return fmt.Errorf("unexpected clip request %+v", post)
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
	post, err := lastClipPost(ctx)
	if err != nil {
		return err
	}
	if post.Preset != "audio" || post.Format != "" {
		return fmt.Errorf("expected the audio preset, sent %+v", post)
	}
	return expectExtension(fileName, ".mp3")
}

func expectSummary(ctx context.Context, want string) error {
	got, err := chromedp.Run(ctx, chromedp.Text(formatSummary))
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("expected format %q, got %q", want, got)
	}
	return nil
}

func testAdvancedClip(ctx context.Context) error {
	if err := openVideo(ctx); err != nil {
		return err
	}
	if err := setRange(ctx, "10", "20"); err != nil {
		return err
	}
	// The format list is only fetched when the panel is opened. Pick formats
	// other than the defaults (137+140) so the choice must take effect.
	if err := chromedp.Do(ctx,
		chromedp.Click(advancedToggle),
		chromedp.WaitReady(videoFormat136),
		chromedp.Click(videoFormat136),
		chromedp.Click(audioFormat251),
	); err != nil {
		return fmt.Errorf("advanced formats did not load: %w", err)
	}
	if err := expectSummary(ctx, "136+251"); err != nil {
		return err
	}

	// A format that already has audio is sent alone and disables the audio list.
	if err := chromedp.Do(ctx, chromedp.Click(videoFormat18)); err != nil {
		return err
	}
	if err := expectSummary(ctx, "18"); err != nil {
		return err
	}
	disabled, err := chromedp.Run(ctx, chromedp.Evaluate[bool](
		`[...document.querySelectorAll("#audioFormats input")].every(i => i.disabled)`))
	if err != nil || !disabled {
		return fmt.Errorf("audio list should be disabled for a format with audio (%v)", err)
	}

	if err := chromedp.Do(ctx, chromedp.Click(videoFormat136)); err != nil {
		return err
	}
	if err := expectSummary(ctx, "136+251"); err != nil {
		return err
	}
	if _, err := clipAndWait(ctx); err != nil {
		return err
	}
	post, err := lastClipPost(ctx)
	if err != nil {
		return err
	}
	if post.Format != "136+251" || post.Preset != "" {
		return fmt.Errorf("expected format 136+251 and no preset, sent %+v", post)
	}
	return nil
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
	// Start 20 is valid against the default end (0:30); the bad end is only
	// submitted with Enter, so the error must come from the submit.
	if err := chromedp.Do(ctx,
		typeTime(fromInput, "20"),
		enterTime(toInput, "10"+kb.Enter),
		chromedp.WaitVisible(timeError),
	); err != nil {
		return fmt.Errorf("no error shown for end before start: %w", err)
	}
	text, err := chromedp.Run(ctx, chromedp.Text(timeError))
	if err != nil || !strings.Contains(text, "after the start") {
		return fmt.Errorf("unexpected time error %q (%v)", text, err)
	}
	posts, err := clipPosts(ctx)
	if err != nil {
		return err
	}
	if len(posts) > 0 {
		return fmt.Errorf("a clip request was sent for an invalid range: %+v", posts)
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
