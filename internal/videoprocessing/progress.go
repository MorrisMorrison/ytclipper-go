package videoprocessing

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/MorrisMorrison/ytclipper-go/internal/jobs"
)

// For --download-sections yt-dlp hands the download to ffmpeg, whose stats
// line ("frame=… time=00:00:12.04 bitrate=… speed=2.41x") ends up on yt-dlp's
// stderr, rewritten in place with \r. time= is how far into the section ffmpeg
// is, so progress is time / clip length; speed= gives the time left.
var (
	ffmpegTimeRe  = regexp.MustCompile(`time=\s*(-?)(\d+):(\d{2}):(\d{2}(?:\.\d+)?)`)
	ffmpegSpeedRe = regexp.MustCompile(`speed=\s*(\d+(?:\.\d+)?)x`)
)

// parseFFmpegStats returns the position (seconds into the output) and speed
// (media seconds per wall-clock second, 0 when unknown) of an ffmpeg stats
// line. ok is false for any other line, or when the time is N/A.
func parseFFmpegStats(line string) (position, speed float64, ok bool) {
	m := ffmpegTimeRe.FindStringSubmatch(line)
	if m == nil {
		return 0, 0, false
	}
	hours, _ := strconv.Atoi(m[2])
	minutes, _ := strconv.Atoi(m[3])
	seconds, _ := strconv.ParseFloat(m[4], 64)
	position = float64(hours*3600+minutes*60) + seconds
	if m[1] == "-" {
		position = 0 // ffmpeg reports a slightly negative time at the start
	}
	if s := ffmpegSpeedRe.FindStringSubmatch(line); s != nil {
		speed, _ = strconv.ParseFloat(s[1], 64)
	}
	return position, speed, true
}

// finishingPrefixes mark yt-dlp's post-processors, which run after the
// download (MP3 conversion, merging, fixups).
var finishingPrefixes = []string{"[ExtractAudio]", "[Merger]", "[Fixup", "[VideoConvertor]", "[VideoRemuxer]"}

// progressTracker turns yt-dlp/ffmpeg output lines into job progress. It is
// safe for concurrent use (stdout and stderr are read by separate goroutines).
type progressTracker struct {
	mu          sync.Mutex
	clipSeconds float64
	percent     int
	finishing   bool
	report      func(phase jobs.Phase, progress, eta *int)
}

func newProgressTracker(clipSeconds float64, report func(phase jobs.Phase, progress, eta *int)) *progressTracker {
	return &progressTracker{clipSeconds: clipSeconds, report: report}
}

// Line handles one output line.
func (t *progressTracker) Line(line string) {
	line = strings.TrimSpace(line)
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.finishing {
		return
	}
	for _, prefix := range finishingPrefixes {
		if strings.HasPrefix(line, prefix) {
			t.finishing = true
			t.percent = 100
			done := 100
			t.report(jobs.PhaseFinishing, &done, nil)
			return
		}
	}

	position, speed, ok := parseFFmpegStats(line)
	if !ok || t.clipSeconds <= 0 {
		return
	}
	position = min(position, t.clipSeconds)
	// Never go backwards: a second ffmpeg run (separate audio, for example)
	// starts again at 0.
	percent := max(t.percent, int(math.Floor(position/t.clipSeconds*100+1e-9)))
	t.percent = percent

	var eta *int
	if speed > 0 {
		left := int(math.Ceil((t.clipSeconds - position) / speed))
		eta = &left
	}
	t.report(jobs.PhaseDownloading, &percent, eta)
}

// lineWriter is an io.Writer that calls fn for every line written to it,
// splitting on \n and on the \r ffmpeg uses to redraw its stats line. A line
// longer than maxLineLength is dropped rather than buffered without bound.
type lineWriter struct {
	buf []byte
	fn  func(string)
}

const maxLineLength = 64 * 1024

func (w *lineWriter) Write(p []byte) (int, error) {
	for _, b := range p {
		if b == '\n' || b == '\r' {
			if len(w.buf) > 0 {
				w.fn(string(w.buf))
				w.buf = w.buf[:0]
			}
			continue
		}
		if len(w.buf) < maxLineLength {
			w.buf = append(w.buf, b)
		}
	}
	return len(p), nil
}
