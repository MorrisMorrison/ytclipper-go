package videoprocessing

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"ytclipper-go/config"

	"github.com/MorrisMorrison/gutils/glogger"
)

const (
	FormatKindVideo = "video" // video only
	FormatKindAudio = "audio" // audio only
	FormatKindBoth  = "both"  // audio and video

	infoCacheTTL     = 10 * time.Minute
	infoCacheMaxSize = 200
)

var videoIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// VideoInfo is what the app needs to know about a video, parsed from
// `yt-dlp -J`. Values returned by GetVideoInfo are shared through the cache:
// treat them as read-only.
type VideoInfo struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Uploader  string   `json:"uploader"`
	Duration  int      `json:"duration"`
	Thumbnail string   `json:"thumbnail"`
	Formats   []Format `json:"formats"`
}

type Format struct {
	ID       string  `json:"id"`
	Ext      string  `json:"ext"`
	Kind     string  `json:"kind"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	FPS      float64 `json:"fps"`
	VCodec   string  `json:"vcodec"`
	ACodec   string  `json:"acodec"`
	TBR      float64 `json:"tbr"`
	ABR      float64 `json:"abr"`
	Filesize int64   `json:"filesize"`
	Note     string  `json:"note"`
}

// Format returns the format with the given id.
func (v *VideoInfo) Format(id string) (Format, bool) {
	for _, f := range v.Formats {
		if f.ID == id {
			return f, true
		}
	}
	return Format{}, false
}

// WatchURL is the canonical URL handed to yt-dlp for a video ID.
func WatchURL(videoID string) string {
	return "https://www.youtube.com/watch?v=" + videoID
}

// GetVideoInfo returns title, duration, thumbnail and formats for a video,
// from the cache when possible. Concurrent calls for the same ID share one
// yt-dlp run. Errors carry a short message that is safe to show to users.
func GetVideoInfo(videoID string) (*VideoInfo, error) {
	if !videoIDPattern.MatchString(videoID) {
		return nil, errors.New("Invalid YouTube video ID")
	}
	return videoInfoCache.get(videoID, func() (*VideoInfo, error) { return fetchVideoInfo(videoID) })
}

// CachedVideoInfo returns the cached info for a video without calling yt-dlp.
func CachedVideoInfo(videoID string) (*VideoInfo, bool) {
	return videoInfoCache.peek(videoID)
}

func fetchVideoInfo(videoID string) (*VideoInfo, error) {
	glogger.Log.Infof("Get Video Info: Fetching info for video %s", videoID)

	output, err := execute("yt-dlp", []string{"-J", "--no-playlist", WatchURL(videoID)})
	if err != nil {
		glogger.Log.Errorf(err, "Get Video Info: Error executing yt-dlp. Output\n%s", string(output))
		return nil, errors.New(summarize(output, err, "Timed out while fetching video information", "Could not fetch video information"))
	}

	if config.CONFIG.Debug {
		glogger.Log.Infof("Get Video Info: yt-dlp command succeeded. Output:\n%s", output)
	}

	info, err := parseVideoInfo(output)
	if err != nil {
		glogger.Log.Errorf(err, "Get Video Info: Could not parse yt-dlp output")
		return nil, errors.New("Could not read video information")
	}
	info.ID = videoID
	return info, nil
}

type rawVideoInfo struct {
	ID        string      `json:"id"`
	Title     string      `json:"title"`
	Uploader  string      `json:"uploader"`
	Channel   string      `json:"channel"`
	Duration  float64     `json:"duration"`
	Thumbnail string      `json:"thumbnail"`
	Formats   []rawFormat `json:"formats"`
}

type rawFormat struct {
	FormatID       string  `json:"format_id"`
	Ext            string  `json:"ext"`
	Width          float64 `json:"width"`
	Height         float64 `json:"height"`
	FPS            float64 `json:"fps"`
	VCodec         string  `json:"vcodec"`
	ACodec         string  `json:"acodec"`
	TBR            float64 `json:"tbr"`
	ABR            float64 `json:"abr"`
	Filesize       float64 `json:"filesize"`
	FilesizeApprox float64 `json:"filesize_approx"`
	FormatNote     string  `json:"format_note"`
	HasDRM         any     `json:"has_drm"`
}

// parseVideoInfo decodes `yt-dlp -J` output. The output is combined
// stdout/stderr, so it skips anything before the line the JSON starts on.
func parseVideoInfo(output []byte) (*VideoInfo, error) {
	start := -1
	if bytes.HasPrefix(output, []byte("{")) {
		start = 0
	} else if i := bytes.Index(output, []byte("\n{")); i >= 0 {
		start = i + 1
	}
	if start < 0 {
		return nil, errors.New("no JSON object in yt-dlp output")
	}

	var raw rawVideoInfo
	if err := json.NewDecoder(bytes.NewReader(output[start:])).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode yt-dlp JSON: %w", err)
	}

	info := &VideoInfo{
		ID:        raw.ID,
		Title:     raw.Title,
		Uploader:  raw.Uploader,
		Duration:  int(math.Round(raw.Duration)),
		Thumbnail: raw.Thumbnail,
		Formats:   make([]Format, 0, len(raw.Formats)),
	}
	if info.Uploader == "" {
		info.Uploader = raw.Channel
	}

	for _, rf := range raw.Formats {
		if f, ok := convertFormat(rf); ok {
			info.Formats = append(info.Formats, f)
		}
	}
	sortFormats(info.Formats)
	return info, nil
}

func codec(c string) string {
	if c == "none" {
		return ""
	}
	return c
}

func convertFormat(rf rawFormat) (Format, bool) {
	if rf.FormatID == "" || rf.Ext == "mhtml" || strings.HasPrefix(rf.FormatID, "sb") {
		return Format{}, false // storyboard
	}
	if drm, _ := rf.HasDRM.(bool); drm {
		return Format{}, false
	}

	vcodec, acodec := codec(rf.VCodec), codec(rf.ACodec)
	var kind string
	switch {
	case vcodec != "" && acodec != "":
		kind = FormatKindBoth
	case vcodec != "":
		kind = FormatKindVideo
	case acodec != "":
		kind = FormatKindAudio
	default:
		return Format{}, false
	}

	size := rf.Filesize
	if size == 0 {
		size = rf.FilesizeApprox
	}

	return Format{
		ID:       rf.FormatID,
		Ext:      rf.Ext,
		Kind:     kind,
		Width:    int(rf.Width),
		Height:   int(rf.Height),
		FPS:      rf.FPS,
		VCodec:   vcodec,
		ACodec:   acodec,
		TBR:      rf.TBR,
		ABR:      rf.ABR,
		Filesize: int64(size),
		Note:     rf.FormatNote,
	}, true
}

// sortFormats puts video formats (with or without audio) first, by height
// then bitrate, followed by audio-only formats by audio bitrate.
func sortFormats(formats []Format) {
	sort.SliceStable(formats, func(i, j int) bool {
		a, b := formats[i], formats[j]
		aAudio, bAudio := a.Kind == FormatKindAudio, b.Kind == FormatKindAudio
		if aAudio != bAudio {
			return !aAudio
		}
		if aAudio {
			return a.ABR > b.ABR
		}
		if a.Height != b.Height {
			return a.Height > b.Height
		}
		return a.TBR > b.TBR
	})
}

var videoInfoCache = newInfoCache(infoCacheTTL, infoCacheMaxSize)

// infoCache keeps video info per video ID for a while. Concurrent lookups for
// an ID that is being fetched wait for that fetch instead of starting another.
// Failed fetches are not cached.
type infoCache struct {
	mu      sync.Mutex
	entries map[string]*infoEntry
	ttl     time.Duration
	maxSize int
	now     func() time.Time
}

type infoEntry struct {
	done    chan struct{} // closed when the fetch finished
	info    *VideoInfo
	err     error
	expires time.Time
}

func newInfoCache(ttl time.Duration, maxSize int) *infoCache {
	return &infoCache{entries: make(map[string]*infoEntry), ttl: ttl, maxSize: maxSize, now: time.Now}
}

func (c *infoCache) get(key string, fetch func() (*VideoInfo, error)) (*VideoInfo, error) {
	c.mu.Lock()
	if e, ok := c.entries[key]; ok {
		select {
		case <-e.done:
			if c.now().Before(e.expires) {
				c.mu.Unlock()
				return e.info, nil
			}
			delete(c.entries, key)
		default:
			c.mu.Unlock()
			<-e.done
			return e.info, e.err
		}
	}

	e := &infoEntry{done: make(chan struct{})}
	c.makeRoomLocked()
	c.entries[key] = e
	c.mu.Unlock()

	info, err := fetch()

	c.mu.Lock()
	e.info, e.err = info, err
	if err != nil {
		delete(c.entries, key)
	} else {
		e.expires = c.now().Add(c.ttl)
	}
	close(e.done)
	c.mu.Unlock()

	return info, err
}

func (c *infoCache) peek(key string) (*VideoInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	select {
	case <-e.done:
		if c.now().Before(e.expires) {
			return e.info, true
		}
		return nil, false
	default:
		return nil, false
	}
}

// makeRoomLocked drops expired entries and, if the cache is still full, the
// finished entry closest to expiry. In-flight fetches are never dropped.
func (c *infoCache) makeRoomLocked() {
	if len(c.entries) < c.maxSize {
		return
	}
	now := c.now()
	oldestKey := ""
	var oldest time.Time
	for k, e := range c.entries {
		select {
		case <-e.done:
		default:
			continue
		}
		if !now.Before(e.expires) {
			delete(c.entries, k)
			continue
		}
		if oldestKey == "" || e.expires.Before(oldest) {
			oldestKey, oldest = k, e.expires
		}
	}
	if len(c.entries) >= c.maxSize && oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}
