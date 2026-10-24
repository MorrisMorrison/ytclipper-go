// Command yt-dlp is a deterministic, offline stand-in for yt-dlp, used by the
// e2e suite (test/e2e). It answers the few invocations the app makes:
//
//	yt-dlp --version        -> a fixed version string
//	yt-dlp -J <url>         -> fixed video info JSON ("Stub Video", 30 s,
//	                           formats 137/136 video only, 18 both, 140/251 audio)
//	yt-dlp -o <tpl> ... url -> writes a small file; %(ext)s becomes mp3 with -x,
//	                           else the --merge-output-format value, else mp4
//
// Every other flag the app passes (--proxy, -f, --download-sections,
// --max-filesize, --downloader-args, ...) is accepted and ignored. URLs that
// are not YouTube links are rejected the way yt-dlp does.
//
// Magic video IDs switch on failure modes, so one shared server can test them:
//
//	failfailfai  the download fails with "ERROR: [youtube] <id>: Video unavailable"
//	slowslowslo  the download takes FAKE_YTDLP_SLOW_SECONDS (default 4)
//	slowinfoslo  -J takes FAKE_YTDLP_SLOW_SECONDS
//	infofailinf  -J fails with "ERROR: [youtube] <id>: Video unavailable"
//	notitlenoti  -J reports an empty title
//
// Environment, set by the harness per server:
//
//	FAKE_YTDLP_TITLE     title for every other video (default "Stub Video")
//	FAKE_YTDLP_UPLOADER  uploader (default "Stub Channel")
//	FAKE_YTDLP_DURATION  duration in seconds (default 30)
package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	idFailDownload = "failfailfai"
	idSlowDownload = "slowslowslo"
	idSlowInfo     = "slowinfoslo"
	idFailInfo     = "infofailinf"
	idNoTitle      = "notitlenoti"
)

// flagsWithValue are the flags the app passes that take a value.
var flagsWithValue = map[string]bool{
	"-o": true, "--output": true,
	"-f": true, "--format": true,
	"--merge-output-format": true,
	"--audio-format":        true,
	"--audio-quality":       true,
	"--proxy":               true,
	"--extractor-retries":   true,
	"--max-filesize":        true,
	"--downloader-args":     true,
	"--download-sections":   true,
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	var (
		info, extractAudio bool
		output, mergeExt   string
		positional         []string
	)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--version", "-U", "--update":
			fmt.Println("ytclipper-fake 0.0.0")
			return 0
		case "-J", "--dump-single-json":
			info = true
			continue
		case "-x", "--extract-audio":
			extractAudio = true
			continue
		}
		if flagsWithValue[a] {
			if i+1 < len(args) {
				switch a {
				case "-o", "--output":
					output = args[i+1]
				case "--merge-output-format":
					mergeExt = args[i+1]
				}
			}
			i++
			continue
		}
		if !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
		}
	}

	if len(positional) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: yt-dlp [OPTIONS] URL [URL...]")
		fmt.Fprintln(os.Stderr, "yt-dlp: error: You must provide at least one URL.")
		return 2
	}
	rawURL := positional[len(positional)-1]
	id, ok := videoID(rawURL)
	if !ok {
		fmt.Fprintf(os.Stderr, "ERROR: [generic] Unsupported URL: %s\n", rawURL)
		return 1
	}

	if info {
		return printInfo(id)
	}
	return download(id, output, extractAudio, mergeExt)
}

// videoID returns the video ID of a watch, youtu.be or Shorts link.
func videoID(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", false
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	path := strings.Trim(u.Path, "/")
	var id string
	switch {
	case host == "youtu.be":
		id = path
	case host == "youtube.com" || host == "m.youtube.com" || host == "music.youtube.com":
		if path == "watch" {
			id = u.Query().Get("v")
		} else if rest, ok := strings.CutPrefix(path, "shorts/"); ok {
			id = rest
		} else if rest, ok := strings.CutPrefix(path, "live/"); ok {
			id = rest
		}
	}
	return id, id != "" && !strings.Contains(id, "/")
}

func slow() {
	seconds := 4.0
	if v, err := strconv.ParseFloat(os.Getenv("FAKE_YTDLP_SLOW_SECONDS"), 64); err == nil && v >= 0 {
		seconds = v
	}
	time.Sleep(time.Duration(seconds * float64(time.Second)))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type format struct {
	FormatID       string   `json:"format_id"`
	FormatNote     string   `json:"format_note"`
	Ext            string   `json:"ext"`
	ACodec         string   `json:"acodec"`
	VCodec         string   `json:"vcodec"`
	Width          *int     `json:"width"`
	Height         *int     `json:"height"`
	FPS            *float64 `json:"fps,omitempty"`
	TBR            float64  `json:"tbr,omitempty"`
	ABR            float64  `json:"abr,omitempty"`
	Filesize       int64    `json:"filesize,omitempty"`
	FilesizeApprox int64    `json:"filesize_approx,omitempty"`
}

func ptr[T any](v T) *T { return &v }

// formats is the fixed format list: a storyboard (ignored by the app), two
// audio-only, one combined and two video-only formats.
var formats = []format{
	{FormatID: "sb0", FormatNote: "storyboard", Ext: "mhtml", ACodec: "none", VCodec: "none", Width: ptr(160), Height: ptr(90)},
	{FormatID: "251", FormatNote: "medium", Ext: "webm", ACodec: "opus", VCodec: "none", TBR: 135.6, ABR: 135.6, Filesize: 508000},
	{FormatID: "140", FormatNote: "medium", Ext: "m4a", ACodec: "mp4a.40.2", VCodec: "none", TBR: 129.5, ABR: 129.5, Filesize: 485000},
	{FormatID: "18", FormatNote: "360p", Ext: "mp4", ACodec: "mp4a.40.2", VCodec: "avc1.42001E", Width: ptr(640), Height: ptr(360), FPS: ptr(30.0), TBR: 492.5, FilesizeApprox: 1850000},
	{FormatID: "136", FormatNote: "720p", Ext: "mp4", ACodec: "none", VCodec: "avc1.4d401f", Width: ptr(1280), Height: ptr(720), FPS: ptr(30.0), TBR: 1150.2, Filesize: 4310000},
	{FormatID: "137", FormatNote: "1080p", Ext: "mp4", ACodec: "none", VCodec: "avc1.640028", Width: ptr(1920), Height: ptr(1080), FPS: ptr(30.0), TBR: 4400.5, Filesize: 16500000},
}

func printInfo(id string) int {
	switch id {
	case idSlowInfo:
		slow()
	case idFailInfo:
		fmt.Fprintf(os.Stderr, "ERROR: [youtube] %s: Video unavailable\n", id)
		return 1
	}
	title := env("FAKE_YTDLP_TITLE", "Stub Video")
	if id == idNoTitle {
		title = ""
	}
	duration, err := strconv.ParseFloat(os.Getenv("FAKE_YTDLP_DURATION"), 64)
	if err != nil || duration <= 0 {
		duration = 30
	}
	out, err := json.Marshal(map[string]any{
		"id":        id,
		"title":     title,
		"uploader":  env("FAKE_YTDLP_UPLOADER", "Stub Channel"),
		"duration":  duration,
		"thumbnail": "https://i.ytimg.com/vi/" + id + "/hqdefault.jpg",
		"formats":   formats,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		return 1
	}
	fmt.Println(string(out))
	return 0
}

func download(id, output string, extractAudio bool, mergeExt string) int {
	switch id {
	case idFailDownload:
		fmt.Fprintf(os.Stderr, "ERROR: [youtube] %s: Video unavailable\n", id)
		return 1
	case idSlowDownload:
		slow()
	}
	if output == "" {
		return 0
	}
	ext := "mp4"
	if extractAudio {
		ext = "mp3"
	} else if mergeExt != "" {
		ext = mergeExt
	}
	path := strings.ReplaceAll(output, "%(ext)s", ext)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		return 1
	}
	// A small payload is enough for the app to offer a download.
	data := make([]byte, 64*1024)
	_, _ = rand.Read(data)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		return 1
	}
	return 0
}
