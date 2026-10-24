package videoprocessing

import (
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// infoFixture is trimmed `yt-dlp -J --no-playlist` output for a real video:
// most fields and formats removed, the shapes kept (nulls, "none" codecs,
// storyboards, filesize_approx, a DRM format). It starts with a stray stderr
// line because the app reads combined output.
const infoFixture = `[youtube] Extracting URL: https://www.youtube.com/watch?v=dQw4w9WgXcQ
{"id": "dQw4w9WgXcQ", "title": "Never Gonna Give You Up", "uploader": "Rick Astley", "channel": "Rick Astley", "duration": 212.0, "thumbnail": "https://i.ytimg.com/vi/dQw4w9WgXcQ/maxresdefault.jpg", "webpage_url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
"formats": [
{"format_id": "sb0", "format_note": "storyboard", "ext": "mhtml", "protocol": "mhtml", "acodec": "none", "vcodec": "none", "width": 160, "height": 90, "fps": 0.5, "tbr": null, "filesize": null},
{"format_id": "139", "format_note": "low", "ext": "m4a", "acodec": "mp4a.40.5", "vcodec": "none", "width": null, "height": null, "fps": null, "tbr": 48.8, "abr": 48.8, "filesize": 1294727, "has_drm": false},
{"format_id": "251", "format_note": "medium", "ext": "webm", "acodec": "opus", "vcodec": "none", "width": null, "height": null, "fps": null, "tbr": 135.6, "abr": 135.6, "filesize": 3596374, "has_drm": false},
{"format_id": "140", "format_note": "medium", "ext": "m4a", "acodec": "mp4a.40.2", "vcodec": "none", "width": null, "height": null, "fps": null, "tbr": 129.5, "abr": 129.5, "filesize": 3433514, "has_drm": false},
{"format_id": "18", "format_note": "360p", "ext": "mp4", "acodec": "mp4a.40.2", "vcodec": "avc1.42001E", "width": 640, "height": 360, "fps": 25, "tbr": 492.5, "abr": null, "filesize": null, "filesize_approx": 13064906, "has_drm": false},
{"format_id": "136", "format_note": "720p", "ext": "mp4", "acodec": "none", "vcodec": "avc1.4d401f", "width": 1280, "height": 720, "fps": 25, "tbr": 1150.2, "vbr": 1150.2, "filesize": 30509012, "has_drm": false},
{"format_id": "247", "format_note": "720p", "ext": "webm", "acodec": "none", "vcodec": "vp9", "width": 1280, "height": 720, "fps": 25, "tbr": 1305.8, "vbr": 1305.8, "filesize": 34637460, "has_drm": false},
{"format_id": "137", "format_note": "1080p", "ext": "mp4", "acodec": "none", "vcodec": "avc1.640028", "width": 1920, "height": 1080, "fps": 25, "tbr": 4400.5, "vbr": 4400.5, "filesize": 116730003, "has_drm": false},
{"format_id": "999", "format_note": "1080p", "ext": "mp4", "acodec": "none", "vcodec": "avc1.640028", "width": 1920, "height": 1080, "fps": 25, "tbr": 5000, "filesize": 1, "has_drm": true}
]}
`

func TestParseVideoInfo(t *testing.T) {
	info, err := parseVideoInfo([]byte(infoFixture))
	if err != nil {
		t.Fatal(err)
	}

	if info.Title != "Never Gonna Give You Up" || info.Uploader != "Rick Astley" || info.Duration != 212 {
		t.Errorf("unexpected metadata %+v", info)
	}
	if info.Thumbnail != "https://i.ytimg.com/vi/dQw4w9WgXcQ/maxresdefault.jpg" {
		t.Errorf("unexpected thumbnail %q", info.Thumbnail)
	}

	var ids []string
	for _, f := range info.Formats {
		ids = append(ids, f.ID)
	}
	// No storyboard, no DRM; video by height then bitrate, then audio by bitrate.
	want := []string{"137", "247", "136", "18", "251", "140", "139"}
	if !slices.Equal(ids, want) {
		t.Errorf("format order = %v; want %v", ids, want)
	}

	f137, _ := info.Format("137")
	if f137 != (Format{ID: "137", Ext: "mp4", Kind: FormatKindVideo, Width: 1920, Height: 1080, FPS: 25, VCodec: "avc1.640028", ACodec: "", TBR: 4400.5, Filesize: 116730003, Note: "1080p"}) {
		t.Errorf("unexpected 137: %+v", f137)
	}
	f18, _ := info.Format("18")
	if f18.Kind != FormatKindBoth || f18.Filesize != 13064906 {
		t.Errorf("18 should be both with the approximate size: %+v", f18)
	}
	f140, _ := info.Format("140")
	if f140.Kind != FormatKindAudio || f140.VCodec != "" || f140.ABR != 129.5 || f140.Width != 0 {
		t.Errorf("unexpected 140: %+v", f140)
	}
}

func TestParseVideoInfoRejectsGarbage(t *testing.T) {
	for _, out := range []string{"", "ERROR: nope", "{not json"} {
		if _, err := parseVideoInfo([]byte(out)); err == nil {
			t.Errorf("parseVideoInfo(%q) succeeded", out)
		}
	}
}

func TestGetVideoInfo(t *testing.T) {
	useFreshInfoCache(t)
	calls := fakeYtDlp(t, "info", infoFixture)

	info, err := GetVideoInfo("dQw4w9WgXcQ")
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != "dQw4w9WgXcQ" || len(info.Formats) != 7 {
		t.Errorf("unexpected info %+v", info)
	}

	args := (*calls)[0]
	if args[0] != "yt-dlp" || !hasFlag(args, "-J") || !hasFlag(args, "--no-playlist") || !hasFlag(args, "--no-warnings") {
		t.Errorf("unexpected args %v", args)
	}
	if args[len(args)-1] != "https://www.youtube.com/watch?v=dQw4w9WgXcQ" {
		t.Errorf("expected the canonical URL last, got %v", args)
	}

	if _, err := GetVideoInfo("dQw4w9WgXcQ"); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 1 {
		t.Errorf("second lookup should hit the cache, got %d calls", len(*calls))
	}
	if cached, ok := CachedVideoInfo("dQw4w9WgXcQ"); !ok || cached != info {
		t.Error("CachedVideoInfo should return the cached info")
	}
}

func TestGetVideoInfoErrors(t *testing.T) {
	useFreshInfoCache(t)
	calls := fakeYtDlp(t, "fail", "[youtube] x\nERROR: [youtube] dQw4w9WgXcQ: Video unavailable")

	if _, err := GetVideoInfo("not-an-id"); err == nil {
		t.Error("expected an error for an invalid ID")
	}
	if len(*calls) != 0 {
		t.Error("invalid IDs must not reach yt-dlp")
	}

	_, err := GetVideoInfo("dQw4w9WgXcQ")
	if err == nil || err.Error() != "[youtube] dQw4w9WgXcQ: Video unavailable" {
		t.Errorf("unexpected error %v", err)
	}
	if _, ok := CachedVideoInfo("dQw4w9WgXcQ"); ok {
		t.Error("failures must not be cached")
	}
}

func TestInfoCacheExpires(t *testing.T) {
	now := time.Now()
	c := newInfoCache(time.Minute, 10)
	c.now = func() time.Time { return now }

	var fetches int
	fetch := func() (*VideoInfo, error) { fetches++; return &VideoInfo{Title: "t"}, nil }

	c.get("a", fetch)
	c.get("a", fetch)
	if fetches != 1 {
		t.Fatalf("fetches = %d; want 1", fetches)
	}

	now = now.Add(2 * time.Minute)
	if _, ok := c.peek("a"); ok {
		t.Error("peek returned an expired entry")
	}
	c.get("a", fetch)
	if fetches != 2 {
		t.Errorf("fetches = %d; want 2 after expiry", fetches)
	}
}

func TestInfoCacheDoesNotCacheErrors(t *testing.T) {
	c := newInfoCache(time.Minute, 10)
	var fetches int
	fetch := func() (*VideoInfo, error) { fetches++; return nil, errors.New("boom") }

	if _, err := c.get("a", fetch); err == nil {
		t.Fatal("expected the error")
	}
	c.get("a", fetch)
	if fetches != 2 {
		t.Errorf("fetches = %d; want 2", fetches)
	}
}

func TestInfoCacheIsBounded(t *testing.T) {
	now := time.Now()
	c := newInfoCache(time.Minute, 3)
	c.now = func() time.Time { now = now.Add(time.Second); return now }

	for _, k := range []string{"a", "b", "c", "d", "e"} {
		c.get(k, func() (*VideoInfo, error) { return &VideoInfo{ID: k}, nil })
	}
	if len(c.entries) > 3 {
		t.Errorf("cache holds %d entries; want at most 3", len(c.entries))
	}
	if _, ok := c.peek("e"); !ok {
		t.Error("the newest entry should be cached")
	}
	if _, ok := c.peek("a"); ok {
		t.Error("the oldest entry should have been evicted")
	}
}

func TestInfoCacheSharesConcurrentFetches(t *testing.T) {
	c := newInfoCache(time.Minute, 10)
	release := make(chan struct{})
	var fetches atomic.Int32
	fetch := func() (*VideoInfo, error) {
		fetches.Add(1)
		<-release
		return &VideoInfo{Title: "shared"}, nil
	}

	const n = 20
	var started, done sync.WaitGroup
	results := make([]*VideoInfo, n)
	for i := range n {
		started.Add(1)
		done.Add(1)
		go func() {
			defer done.Done()
			started.Done()
			results[i], _ = c.get("a", fetch)
		}()
	}
	started.Wait()
	time.Sleep(20 * time.Millisecond) // let the goroutines reach get
	close(release)
	done.Wait()

	if got := fetches.Load(); got != 1 {
		t.Errorf("fetches = %d; want 1", got)
	}
	for i, r := range results {
		if r == nil || r.Title != "shared" {
			t.Fatalf("result %d = %+v", i, r)
		}
	}
}
