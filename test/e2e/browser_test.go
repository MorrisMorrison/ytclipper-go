//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/emulation"
	cdppage "github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// testTimeout bounds one test's browser actions.
const testTimeout = 90 * time.Second

// youTubeHosts are blocked in the normal suite: no player or thumbnails, so
// runs stay offline and deterministic and the "player unavailable" path is
// what's tested.
const youTubeHosts = "MAP youtube.com ~NOTFOUND, MAP *.youtube.com ~NOTFOUND, " +
	"MAP youtube-nocookie.com ~NOTFOUND, MAP *.youtube-nocookie.com ~NOTFOUND, MAP *.ytimg.com ~NOTFOUND"

var (
	sharedBrowserOnce sync.Once
	sharedBrowser     context.Context
	sharedBrowserErr  error
)

// startBrowser starts a headless browser that lives until TestMain ends.
// Tests open their own tab, in their own browser context, on it.
func startBrowser(blockYouTube bool) (context.Context, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.WindowSize(1280, 1000),
	)
	if blockYouTube {
		opts = append(opts, chromedp.Flag("host-resolver-rules", youTubeHosts))
	} else {
		opts = append(opts, chromedp.Flag("autoplay-policy", "user-gesture-required"))
	}
	if path := os.Getenv("E2E_CHROME"); path != "" {
		opts = append(opts, chromedp.ExecPath(path))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(format string, args ...any) {
		if msg := fmt.Sprintf(format, args...); !strings.Contains(msg, "could not unmarshal event") {
			log.Printf("chrome: %s", msg)
		}
	}))
	addCleanup(func() {
		cancel()
		cancelAlloc()
	})
	if err := chromedp.Do(ctx); err != nil {
		return nil, fmt.Errorf("starting the browser (set E2E_CHROME to its path if it isn't found): %w", err)
	}
	return ctx, nil
}

func browserFor(t *testing.T) context.Context {
	t.Helper()
	sharedBrowserOnce.Do(func() {
		sharedBrowser, sharedBrowserErr = startBrowser(true)
	})
	if sharedBrowserErr != nil {
		t.Fatal(sharedBrowserErr)
	}
	return sharedBrowser
}

// fetchLogJS records every fetch the page makes in window.__fetches as
// {url, method, body, status}; status stays 0 until the response arrives.
const fetchLogJS = `(() => {
	window.__fetches = [];
	const original = window.fetch;
	window.fetch = function (input, options = {}) {
		const entry = { url: String(typeof input === "string" ? input : input.url), method: (options.method || "GET").toUpperCase(), body: null, status: 0 };
		try { if (options.body) entry.body = JSON.parse(options.body); } catch { entry.body = String(options.body); }
		window.__fetches.push(entry);
		return original.apply(this, arguments).then((response) => { entry.status = response.status; return response; });
	};
})();`

// page is one browser tab, in its own browser context (fresh storage), with
// a fetch log, a console log and failure artifacts.
type page struct {
	t    *testing.T
	ctx  context.Context // the tab, bounded by testTimeout
	tab  context.Context
	base string

	mu      sync.Mutex
	console []string
}

type pageOption func(*pageConfig)

type pageConfig struct {
	browser     context.Context
	base        string
	colorScheme string // "light", "dark" or "" (the browser's)
	width       int64
	height      int64
	mobile      bool
}

func withServer(base string) pageOption { return func(c *pageConfig) { c.base = base } }
func withBrowser(b context.Context) pageOption {
	return func(c *pageConfig) { c.browser = b }
}
func withColorScheme(scheme string) pageOption { return func(c *pageConfig) { c.colorScheme = scheme } }
func withPhone() pageOption {
	return func(c *pageConfig) { c.width, c.height, c.mobile = 375, 812, true }
}
func withViewport(w, h int64) pageOption {
	return func(c *pageConfig) { c.width, c.height = w, h }
}

func newPage(t *testing.T, opts ...pageOption) *page {
	t.Helper()
	cfg := &pageConfig{}
	for _, o := range opts {
		o(cfg)
	}
	if cfg.browser == nil {
		cfg.browser = browserFor(t)
	}
	if cfg.base == "" {
		cfg.base = baseURL(t)
	}

	tab, cancelTab := chromedp.NewContext(cfg.browser, chromedp.WithNewBrowserContext())
	t.Cleanup(cancelTab)
	p := &page{t: t, tab: tab, base: cfg.base}

	messages := chromedp.Console(tab)
	go func() {
		for m, err := range messages {
			if err != nil {
				return
			}
			p.mu.Lock()
			p.console = append(p.console, fmt.Sprintf("%s %s", m.Time.Format("15:04:05.000"), m))
			p.mu.Unlock()
		}
	}()

	// The page starts the download on its own when a clip is ready; don't
	// write files from the test run.
	if _, err := chromedp.CallBrowser(tab, browser.SetDownloadBehavior, browser.SetDownloadBehaviorParams{
		Behavior:         browser.SetDownloadBehaviorBehaviorDeny,
		BrowserContextID: chromedp.FromContext(tab).BrowserContextID,
	}); err != nil {
		t.Fatalf("disabling downloads: %v", err)
	}
	// Parallel tabs are all in the background; let each behave as focused
	// so focus and keyboard events work.
	if _, err := chromedp.Call(tab, emulation.SetFocusEmulationEnabled, emulation.SetFocusEmulationEnabledParams{Enabled: true}); err != nil {
		t.Fatalf("focus emulation: %v", err)
	}
	if _, err := chromedp.Call(tab, cdppage.AddScriptToEvaluateOnNewDocument, cdppage.AddScriptToEvaluateOnNewDocumentParams{Source: fetchLogJS}); err != nil {
		t.Fatalf("installing the fetch log: %v", err)
	}
	if cfg.colorScheme != "" {
		if _, err := chromedp.Call(tab, emulation.SetEmulatedMedia, emulation.SetEmulatedMediaParams{
			Features: []*emulation.MediaFeature{{Name: "prefers-color-scheme", Value: cfg.colorScheme}},
		}); err != nil {
			t.Fatalf("emulating the color scheme: %v", err)
		}
	}
	if cfg.width > 0 {
		var vopts []chromedp.EmulateViewportOption
		if cfg.mobile {
			vopts = append(vopts, chromedp.EmulateMobile, chromedp.EmulateTouch, chromedp.EmulatePortrait)
		}
		if err := chromedp.Do(tab, chromedp.EmulateViewport(cfg.width, cfg.height, vopts...)); err != nil {
			t.Fatalf("emulating the viewport: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(tab, testTimeout)
	t.Cleanup(cancel)
	p.ctx = ctx
	// Runs before the cancels above (cleanups run last in, first out).
	t.Cleanup(p.saveArtifactsOnFailure)
	return p
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// saveArtifactsOnFailure writes a full-page screenshot, the console log and
// the fetch log of a failed test to the artifacts directory.
func (p *page) saveArtifactsOnFailure() {
	if !p.t.Failed() {
		return
	}
	name := unsafeName.ReplaceAllString(p.t.Name(), "_")
	ctx, cancel := context.WithTimeout(p.tab, 15*time.Second)
	defer cancel()

	if png, err := chromedp.Run(ctx, chromedp.FullScreenshot(100)); err == nil {
		path := filepath.Join(artifactsDir, name+".png")
		if err := os.WriteFile(path, png, 0o644); err == nil {
			p.t.Logf("screenshot: %s", path)
		}
	} else {
		p.t.Logf("no screenshot: %v", err)
	}

	p.mu.Lock()
	logText := strings.Join(p.console, "\n")
	p.mu.Unlock()
	if fetches, err := chromedp.Run(ctx, chromedp.Evaluate[[]fetchEntry](`window.__fetches || []`)); err == nil {
		var b strings.Builder
		for _, f := range fetches {
			fmt.Fprintf(&b, "%s %s -> %d %v\n", f.Method, f.URL, f.Status, f.Body)
		}
		logText += "\n\n--- fetches ---\n" + b.String()
	}
	path := filepath.Join(artifactsDir, name+".console.txt")
	if err := os.WriteFile(path, []byte(logText+"\n"), 0o644); err == nil {
		p.t.Logf("console log: %s", path)
	}
}

// ===== actions =====

// do runs actions and fails the test with what on error.
func (p *page) do(what string, actions ...chromedp.Action[chromedp.Void]) {
	p.t.Helper()
	if err := chromedp.Do(p.ctx, actions...); err != nil {
		p.t.Fatalf("%s: %v", what, err)
	}
}

// eval evaluates a JavaScript expression in the page.
func eval[T any](p *page, expression string) T {
	p.t.Helper()
	v, err := chromedp.Run(p.ctx, chromedp.Evaluate[T](expression))
	if err != nil {
		p.t.Fatalf("evaluating %s: %v", expression, err)
	}
	return v
}

// poll waits until the expression is truthy and returns its value.
func poll[T any](p *page, what, expression string, timeout time.Duration) T {
	p.t.Helper()
	ctx, cancel := context.WithTimeout(p.ctx, timeout+5*time.Second)
	defer cancel()
	v, err := chromedp.Run(ctx, chromedp.Poll[T](expression,
		chromedp.WithPollingInterval(100*time.Millisecond), chromedp.WithPollingTimeout(timeout)))
	if err != nil {
		p.t.Fatalf("%s (waited %v): %v", what, timeout, err)
	}
	return v
}

func (p *page) text(sel string) string {
	p.t.Helper()
	v, err := chromedp.Run(p.ctx, chromedp.Text(chromedp.CSS(sel)))
	if err != nil {
		p.t.Fatalf("reading the text of %s: %v", sel, err)
	}
	return v
}

func (p *page) value(sel string) string {
	p.t.Helper()
	return eval[string](p, fmt.Sprintf(`document.querySelector(%q).value`, sel))
}

func (p *page) hidden(id string) bool {
	p.t.Helper()
	return eval[bool](p, fmt.Sprintf(`document.getElementById(%q).hidden`, id))
}

// navigate loads the page.
func (p *page) navigate() {
	p.t.Helper()
	p.do("loading the page", chromedp.Navigate(p.base), chromedp.WaitVisible(urlInput))
}

// openVideo loads the page and enters a link; the editor appears without any
// server round trip. It waits until the video length (from the fake's
// /video/info) is on the slider.
func (p *page) openVideo(link string) {
	p.t.Helper()
	p.enterURL(link)
	poll[bool](p, "the video length did not arrive", `document.getElementById("trimmer").dataset.ready === "true"`, 15*time.Second)
}

// enterURL loads the page, types a link and waits for the editor.
func (p *page) enterURL(link string) {
	p.t.Helper()
	p.navigate()
	p.do("the editor did not appear after entering a URL",
		chromedp.SendKeys(urlInput, link),
		chromedp.WaitVisible(editor),
	)
}

// enterTime focuses a time field, then replaces its value with real
// keystrokes. Focusing first matters: the page doesn't rewrite a focused
// field when the video length arrives.
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

func (p *page) setRange(from, to string) {
	p.t.Helper()
	p.do("typing the time range", typeTime(fromInput, from), typeTime(toInput, to))
}

type fetchEntry struct {
	URL    string `json:"url"`
	Method string `json:"method"`
	Body   any    `json:"body"`
	Status int    `json:"status"`
}

func (p *page) fetches(substr string) []fetchEntry {
	p.t.Helper()
	all := eval[[]fetchEntry](p, `window.__fetches || []`)
	var out []fetchEntry
	for _, f := range all {
		if strings.Contains(f.URL, substr) {
			out = append(out, f)
		}
	}
	return out
}

type clipPayload struct {
	URL    string `json:"url"`
	Preset string `json:"preset"`
	Format string `json:"format"`
	From   string `json:"from"`
	To     string `json:"to"`
}

// clipPosts returns the bodies of every POST /api/v1/clip.
func (p *page) clipPosts() []clipPayload {
	p.t.Helper()
	return eval[[]clipPayload](p, `(window.__fetches || []).filter(f => f.method === "POST" && f.url.endsWith("/api/v1/clip")).map(f => f.body)`)
}

func (p *page) lastClipPost() clipPayload {
	p.t.Helper()
	posts := p.clipPosts()
	if len(posts) == 0 {
		p.t.Fatal("the page sent no clip request")
	}
	return posts[len(posts)-1]
}

// submit clicks "Create clip" and waits for the job to end. It returns the
// final state ("completed" or "error") and the job detail text.
func (p *page) submit() (state, detail string) {
	p.t.Helper()
	p.do("clicking Create clip", chromedp.Click(clipButton))
	state = poll[string](p, "the job did not finish",
		`(s => s === "completed" || s === "error" ? s : false)(document.getElementById("job").dataset.state)`, 40*time.Second)
	return state, p.text("#jobDetail")
}

// clip submits and requires a completed job. It returns the file name the
// server reported and the download link.
func (p *page) clip() (fileName, href string) {
	p.t.Helper()
	state, detail := p.submit()
	if state != "completed" {
		p.t.Fatalf("job failed: %s", detail)
	}
	p.do("download button not shown", chromedp.WaitVisible(downloadButton))
	href = eval[string](p, `document.getElementById("downloadButton").getAttribute("href")`)
	if !strings.Contains(href, "/api/v1/clip?jobId=") {
		p.t.Fatalf("unexpected download link %q", href)
	}
	return detail, href
}

func (p *page) toastCount() int {
	p.t.Helper()
	return eval[int](p, `document.getElementById("toasts").childElementCount`)
}
