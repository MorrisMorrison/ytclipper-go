//go:build e2e

package e2e

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

const (
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
	clipButton     = chromedp.CSS(`#clipButton`)
	downloadButton = chromedp.CSS(`#downloadButton`)
	themeToggle    = chromedp.CSS(`#themeToggle`)
	trimTrack      = chromedp.CSS(`#trimTrack`)
	handleStart    = chromedp.CSS(`#handleStart`)
	handleEnd      = chromedp.CSS(`#handleEnd`)

	// The fake yt-dlp answers every video ID with "Stub Video", 30 s long,
	// except the magic IDs below (see test/fakes/yt-dlp).
	stubVideoID     = "dQw4w9WgXcQ"
	failDownloadID  = "failfailfai"
	slowDownloadID  = "slowslowslo"
	slowInfoID      = "slowinfoslo"
	invalidURL      = "invalid-url"
	serverBusyError = "The server is busy, try again in a minute."
)

func watchURL(id string) string { return "https://www.youtube.com/watch?v=" + id }

func TestVideoPreset(t *testing.T) {
	t.Parallel()
	p := newPage(t)
	p.openVideo(watchURL(stubVideoID))

	// YouTube is unreachable here, so the page must say so and still know the
	// length from /video/info.
	poll[bool](p, "player-unavailable state not shown",
		`document.querySelector(".player-frame").dataset.player === "unavailable"`, 20*time.Second)
	if got := p.text("#trimDuration"); got != "0:30" {
		t.Fatalf("expected the fake's length 0:30 on the slider, got %q", got)
	}
	p.setRange("10", "20")
	fileName, _ := p.clip()

	post := p.lastClipPost()
	if post.Preset != "video" || post.Format != "" || post.From != "00:00:10" || post.To != "00:00:20" || post.URL != watchURL(stubVideoID) {
		t.Fatalf("unexpected clip request %+v", post)
	}
	if !strings.HasSuffix(fileName, ".mp4") {
		t.Fatalf("expected an .mp4 file, got %q", fileName)
	}
}

func TestAudioPreset(t *testing.T) {
	t.Parallel()
	p := newPage(t)
	p.openVideo(watchURL(stubVideoID))
	p.setRange("0:05", "0:15")
	p.do("choosing the audio preset", chromedp.Click(presetAudio))
	fileName, _ := p.clip()

	if post := p.lastClipPost(); post.Preset != "audio" || post.Format != "" {
		t.Fatalf("expected the audio preset, sent %+v", post)
	}
	if !strings.HasSuffix(fileName, ".mp3") {
		t.Fatalf("expected an .mp3 file, got %q", fileName)
	}
}

func TestAdvancedFormats(t *testing.T) {
	t.Parallel()
	p := newPage(t)
	p.openVideo(watchURL(stubVideoID))
	p.setRange("10", "20")

	expectSummary := func(want string) {
		t.Helper()
		if got := p.text("#formatSummary"); got != want {
			t.Fatalf("expected format %q, got %q", want, got)
		}
	}

	// Pick formats other than the defaults (137+140) so the choice must take
	// effect.
	p.do("advanced formats did not load",
		chromedp.Click(advancedToggle),
		chromedp.WaitReady(videoFormat136),
		chromedp.Click(videoFormat136),
		chromedp.Click(audioFormat251),
	)
	expectSummary("136+251")

	// A format that already has audio is sent alone and disables the audio list.
	p.do("choosing format 18", chromedp.Click(videoFormat18))
	expectSummary("18")
	if !eval[bool](p, `[...document.querySelectorAll("#audioFormats input")].every(i => i.disabled)`) {
		t.Fatal("audio list should be disabled for a format with audio")
	}

	p.do("choosing format 136", chromedp.Click(videoFormat136))
	expectSummary("136+251")
	p.clip()
	if post := p.lastClipPost(); post.Format != "136+251" || post.Preset != "" {
		t.Fatalf("expected format 136+251 and no preset, sent %+v", post)
	}
}

func TestInvalidURL(t *testing.T) {
	t.Parallel()
	p := newPage(t)
	p.navigate()
	p.do("no error shown for an invalid URL",
		chromedp.SendKeys(urlInput, invalidURL),
		chromedp.Blur(urlInput),
		chromedp.WaitVisible(urlError),
	)
	if !p.hidden("editor") {
		t.Fatal("editor is shown for an invalid URL")
	}
	if n := len(p.fetches("/api/")); n != 0 {
		t.Fatalf("an invalid URL caused %d API request(s)", n)
	}
}

func TestEndBeforeStart(t *testing.T) {
	t.Parallel()
	p := newPage(t)
	p.openVideo(watchURL(stubVideoID))
	// Start 20 is valid against the default end (0:30); the bad end is only
	// submitted with Enter, so the error must come from the submit.
	p.do("no error shown for end before start",
		typeTime(fromInput, "20"),
		enterTime(toInput, "10"+kb.Enter),
		chromedp.WaitVisible(timeError),
	)
	if text := p.text("#timeError"); !strings.Contains(text, "after the start") {
		t.Fatalf("unexpected time error %q", text)
	}
	if posts := p.clipPosts(); len(posts) > 0 {
		t.Fatalf("a clip request was sent for an invalid range: %+v", posts)
	}
	if !p.hidden("job") {
		t.Fatal("a job was started for an invalid range")
	}
}

// TestDownload fetches the finished clip the way the browser does.
func TestDownload(t *testing.T) {
	t.Parallel()
	p := newPage(t)
	p.openVideo(watchURL(stubVideoID))
	p.setRange("0:10", "0:20")
	fileName, href := p.clip()

	resp, body, err := httpGet(p.base + href)
	if err != nil {
		t.Fatalf("downloading %s: %v", href, err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("download: HTTP %d", resp.StatusCode)
	}
	disposition := resp.Header.Get("Content-Disposition")
	if !strings.HasPrefix(disposition, "attachment;") || !strings.Contains(disposition, "filename*=UTF-8''") {
		t.Fatalf("unexpected Content-Disposition %q", disposition)
	}
	if !strings.Contains(disposition, "Stub%20Video") || !strings.HasSuffix(fileName, ".mp4") {
		t.Fatalf("download name %q / Content-Disposition %q don't match the video", fileName, disposition)
	}
	if len(body) == 0 {
		t.Fatal("the downloaded clip is empty")
	}
}

// TestJobError: yt-dlp's error message ends up in the job panel, and only
// there (no toast, or screen readers would hear it twice).
func TestJobError(t *testing.T) {
	t.Parallel()
	p := newPage(t)
	p.openVideo(watchURL(failDownloadID))
	state, detail := p.submit()
	if state != "error" {
		t.Fatalf("expected the job to fail, got %q (%s)", state, detail)
	}
	if !strings.Contains(detail, "Video unavailable") || strings.Contains(detail, "ERROR:") {
		t.Fatalf("expected yt-dlp's message in the panel, got %q", detail)
	}
	if title := p.text("#jobTitle"); title != "The clip couldn't be created" {
		t.Fatalf("unexpected job title %q", title)
	}
	if n := p.toastCount(); n != 0 {
		t.Fatalf("expected no toast for a job error, got %d", n)
	}
	if !p.hidden("downloadButton") {
		t.Fatal("download button shown for a failed job")
	}
}

// TestQueueFull fills a one-job queue with a slow clip; the next clip request
// gets 429 and the page shows the server's busy message.
func TestQueueFull(t *testing.T) {
	t.Parallel()
	s := ownServer(t, "queue", map[string]string{
		"YTCLIPPER_MAX_CONCURRENT_JOBS": "1",
		"YTCLIPPER_MAX_QUEUED_JOBS":     "1",
		"FAKE_YTDLP_SLOW_SECONDS":       "8",
	})

	first := newPage(t, withServer(s.URL))
	first.openVideo(watchURL(slowDownloadID))
	first.do("clicking Create clip", chromedp.Click(clipButton))
	poll[bool](first, "the slow job was not accepted",
		`(window.__fetches || []).some(f => f.method === "POST" && f.url.endsWith("/api/v1/clip") && f.status === 201)`, 10*time.Second)

	second := newPage(t, withServer(s.URL))
	second.openVideo(watchURL(stubVideoID))
	state, detail := second.submit()
	if state != "error" || detail != serverBusyError {
		t.Fatalf("expected %q, got state %q: %q", serverBusyError, state, detail)
	}
	if posts := second.fetches("/api/v1/clip"); len(posts) != 1 || posts[0].Status != http.StatusTooManyRequests {
		t.Fatalf("expected one clip request answered with 429, got %+v", posts)
	}
	if n := second.toastCount(); n != 0 {
		t.Fatalf("expected no toast, got %d", n)
	}
}

// TestLazyFormats: the editor never waits for /video/info, and the format
// lists are only built once Advanced is opened.
func TestLazyFormats(t *testing.T) {
	t.Parallel()
	p := newPage(t)
	// The fake answers -J for this ID after a few seconds.
	p.enterURL(watchURL(slowInfoID))

	infos := p.fetches("/api/v1/video/info?")
	if len(infos) != 1 || !strings.Contains(infos[0].URL, slowInfoID) {
		t.Fatalf("expected one video info request, got %+v", infos)
	}
	if infos[0].Status != 0 {
		t.Fatalf("the editor waited for the video info (status %d already)", infos[0].Status)
	}

	poll[bool](p, "the video info did not arrive",
		`(window.__fetches || []).some(f => f.url.includes("/api/v1/video/info?") && f.status === 200)`, 20*time.Second)
	if n := eval[int](p, `document.querySelectorAll("#videoFormats input, #audioFormats input").length`); n != 0 {
		t.Fatalf("format lists were built before Advanced was opened (%d inputs)", n)
	}
	if !p.hidden("advancedContent") {
		t.Fatal("advanced content shown before Advanced was opened")
	}

	p.do("advanced formats did not load", chromedp.Click(advancedToggle), chromedp.WaitVisible(videoFormat136))
	if n := eval[int](p, `document.querySelectorAll("#videoFormats input").length`); n != 3 {
		t.Fatalf("expected the fake's 3 video formats (137, 136, 18), got %d", n)
	}
	if n := len(p.fetches("/api/v1/video/info?")); n != 1 {
		t.Fatalf("opening Advanced fetched the video info again (%d requests)", n)
	}
}

type point struct{ X, Y float64 }

type trimGeometry struct {
	Left  float64 `json:"left"`
	Width float64 `json:"width"`
	Y     float64 `json:"y"`
	Start point   `json:"start"`
	End   point   `json:"end"`
}

func (p *page) trimGeometry() trimGeometry {
	p.t.Helper()
	return eval[trimGeometry](p, `(() => {
		const track = document.getElementById("trimTrack").getBoundingClientRect();
		const center = (id) => { const r = document.getElementById(id).getBoundingClientRect(); return { X: r.left + r.width / 2, Y: r.top + r.height / 2 }; };
		return { left: track.left, width: track.width, y: track.top + track.height / 2, start: center("handleStart"), end: center("handleEnd") };
	})()`)
}

// drag presses at from, moves to to in steps and releases, as a mouse does.
func (p *page) drag(from point, toX float64) {
	p.t.Helper()
	held := func(e *input.DispatchMouseEventParams) {
		e.Button = input.MouseButtonLeft
		e.Buttons = 1
		e.ClickCount = 1
	}
	steps := []chromedp.Action[chromedp.Void]{chromedp.MouseEvent(chromedp.MousePressed, from.X, from.Y, held)}
	const n = 8
	for i := 1; i <= n; i++ {
		x := from.X + (toX-from.X)*float64(i)/n
		steps = append(steps, chromedp.MouseEvent(chromedp.MouseMoved, x, from.Y, held))
	}
	steps = append(steps, chromedp.MouseEvent(chromedp.MouseReleased, toX, from.Y, held))
	p.do("dragging a trimmer handle", steps...)
}

func (p *page) expectFields(from, to string) {
	p.t.Helper()
	if gotFrom, gotTo := p.value("#from"), p.value("#to"); gotFrom != from || gotTo != to {
		p.t.Fatalf("expected the fields %s–%s, got %s–%s", from, to, gotFrom, gotTo)
	}
}

func (p *page) handleValues() (start, end int) {
	p.t.Helper()
	v := eval[[]int](p, `["handleStart", "handleEnd"].map(id => Number(document.getElementById(id).getAttribute("aria-valuenow")))`)
	return v[0], v[1]
}

func TestTrimmer(t *testing.T) {
	t.Parallel()
	p := newPage(t)
	p.openVideo(watchURL(stubVideoID))
	p.do("scrolling to the trimmer", chromedp.ScrollIntoView(trimTrack))
	at := func(g trimGeometry, seconds float64) float64 { return g.Left + g.Width*seconds/30 }

	// Drag both handles (the fake's video is 30 s long).
	g := p.trimGeometry()
	p.drag(g.Start, at(g, 10))
	p.expectFields("0:10", "0:30")
	g = p.trimGeometry()
	p.drag(g.End, at(g, 20))
	p.expectFields("0:10", "0:20")
	if got := p.text("#clipLength"); got != "0:10" {
		t.Fatalf("expected the length 0:10, got %q", got)
	}

	// Keyboard: arrows move 1 s, Shift+arrows 10 s.
	p.do("pressing ArrowRight on the start handle", chromedp.Focus(handleStart), chromedp.KeyEvent(kb.ArrowRight))
	p.expectFields("0:11", "0:20")
	p.do("pressing Shift+ArrowRight on the end handle", chromedp.Focus(handleEnd),
		chromedp.KeyEvent(kb.ArrowRight, chromedp.KeyModifiers(chromedp.ModifierShift)))
	p.expectFields("0:11", "0:30")

	// Typed times move the handles.
	p.setRange("5", "25")
	if start, end := p.handleValues(); start != 5 || end != 25 {
		t.Fatalf("expected the handles at 5 and 25 s, got %d and %d", start, end)
	}
	left := eval[float64](p, `parseFloat(document.getElementById("handleStart").style.left)`)
	if math.Abs(left-100.0/6) > 0.1 {
		t.Fatalf("expected the start handle at 16.67%%, got %.2f%%", left)
	}

	// The handles can't cross: the end stops one second after the start, and
	// the start one second before the end.
	g = p.trimGeometry()
	p.drag(g.End, at(g, 1))
	if start, end := p.handleValues(); start != 5 || end != 6 {
		t.Fatalf("expected the end to stop at 6 s, got %d–%d", start, end)
	}
	p.expectFields("0:05", "0:06")
	p.setRange("5", "25")
	g = p.trimGeometry()
	p.drag(g.Start, at(g, 29))
	if start, end := p.handleValues(); start != 24 || end != 25 {
		t.Fatalf("expected the start to stop at 24 s, got %d–%d", start, end)
	}
	p.expectFields("0:24", "0:25")
}

func TestURLVariants(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, link, id string }{
		{"youtu.be", "https://youtu.be/abcdefghijk?si=AbCdEf123", "abcdefghijk"},
		{"shorts", "https://www.youtube.com/shorts/ABCDEFGHIJK", "ABCDEFGHIJK"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := newPage(t)
			p.openVideo(tc.link)
			infos := p.fetches("/api/v1/video/info?")
			if len(infos) != 1 || !strings.Contains(infos[0].URL, "watch%3Fv%3D"+tc.id) {
				t.Fatalf("expected one info request for %s, got %+v", tc.id, infos)
			}
			if title := p.text("#videoTitle"); title != "Stub Video" {
				t.Fatalf("expected the title Stub Video, got %q", title)
			}
		})
	}
}

func TestPhoneViewport(t *testing.T) {
	t.Parallel()
	p := newPage(t, withPhone())
	p.openVideo(watchURL(stubVideoID))

	overflow := func(when string) {
		t.Helper()
		if w := eval[[]int](p, `[document.documentElement.scrollWidth, window.innerWidth]`); w[0] > w[1] {
			t.Fatalf("%s: the page is %d px wide in a %d px viewport", when, w[0], w[1])
		}
	}
	overflow("simple mode")
	p.do("opening Advanced", chromedp.Click(advancedToggle), chromedp.WaitVisible(videoFormat136))
	overflow("advanced mode")

	p.do("scrolling to Create clip", chromedp.ScrollIntoView(clipButton))
	visible := eval[bool](p, `(() => {
		const r = document.getElementById("clipButton").getBoundingClientRect();
		return r.width > 0 && r.height > 0 && r.top >= 0 && r.bottom <= window.innerHeight && r.left >= 0 && r.right <= window.innerWidth;
	})()`)
	if !visible {
		t.Fatal("Create clip is not fully visible after scrolling to it")
	}
}

func TestThemeToggle(t *testing.T) {
	t.Parallel()
	p := newPage(t, withColorScheme("light"))
	p.navigate()

	theme := func() string {
		t.Helper()
		return eval[string](p, `document.documentElement.getAttribute("data-theme")`)
	}
	if got := theme(); got != "light" {
		t.Fatalf("expected the OS theme light, got %q", got)
	}
	for _, want := range []string{"dark", "light"} {
		p.do("clicking the theme toggle", chromedp.Click(themeToggle))
		if got := theme(); got != want {
			t.Fatalf("expected %s after the toggle, got %q", want, got)
		}
		if pressed := eval[string](p, `document.getElementById("themeToggle").getAttribute("aria-pressed")`); pressed != fmt.Sprint(want == "dark") {
			t.Fatalf("aria-pressed is %q with the %s theme", pressed, want)
		}
		p.do("reloading", chromedp.Reload(), chromedp.WaitVisible(urlInput))
		if got := theme(); got != want {
			t.Fatalf("the %s theme did not survive a reload (got %q)", want, got)
		}
	}
}
