package videoprocessing

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MorrisMorrison/ytclipper-go/internal/jobs"
)

func TestParseFFmpegStats(t *testing.T) {
	tests := []struct {
		line            string
		position, speed float64
		ok              bool
	}{
		{"frame=  361 fps= 72 q=-1.0 size=    2048KiB time=00:00:12.04 bitrate=1393.5kbits/s speed=2.41x    ", 12.04, 2.41, true},
		{"size=     256KiB time=00:01:16.32 bitrate= 128.5kbits/s speed=32.6x", 76.32, 32.6, true},
		{"size=       0KiB time=01:02:03.50 bitrate=N/A speed=N/A", 3723.5, 0, true},
		{"frame=    0 fps=0.0 q=0.0 size=       0KiB time=-00:00:00.02 bitrate=N/A speed=N/A", 0, 0, true},
		{"size=N/A time=N/A bitrate=N/A speed=N/A", 0, 0, false},
		{"[download] Destination: /tmp/x.mp4", 0, 0, false},
		{"", 0, 0, false},
	}
	for _, tt := range tests {
		position, speed, ok := parseFFmpegStats(tt.line)
		if ok != tt.ok || position != tt.position || speed != tt.speed {
			t.Errorf("parseFFmpegStats(%q) = %v, %v, %v; want %v, %v, %v", tt.line, position, speed, ok, tt.position, tt.speed, tt.ok)
		}
	}
}

type progressReport struct {
	phase         jobs.Phase
	progress, eta string
}

func show(v *int) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprint(*v)
}

func recordingTracker(clipSeconds float64) (*progressTracker, *[]progressReport) {
	reports := &[]progressReport{}
	tracker := newProgressTracker(clipSeconds, func(phase jobs.Phase, progress, eta *int) {
		*reports = append(*reports, progressReport{phase, show(progress), show(eta)})
	})
	return tracker, reports
}

func TestProgressTracker(t *testing.T) {
	tracker, reports := recordingTracker(30)
	for _, line := range []string{
		"[youtube] dQw4w9WgXcQ: Downloading webpage",
		"[download] Destination: /clips/x.webm",
		"frame=  100 fps=0.0 q=-1.0 size=  512KiB time=00:00:03.00 bitrate=1398.1kbits/s speed=6.0x",
		"frame=  361 fps= 72 q=-1.0 size= 2048KiB time=00:00:12.60 bitrate=1393.5kbits/s speed=2.0x",
		"size=  128KiB time=00:00:01.00 bitrate=1048.6kbits/s speed=N/A", // second run: never backwards
		"[ExtractAudio] Destination: /clips/x.mp3",
		"size=  128KiB time=00:00:20.00 bitrate=52.4kbits/s speed=40x", // ignored once finishing
		"[Merger] Merging formats into \"/clips/x.mp4\"",
	} {
		tracker.Line(line)
	}

	want := []progressReport{
		{jobs.PhaseDownloading, "10", "5"},   // 3/30; (30-3)/6.0 = 4.5 → 5
		{jobs.PhaseDownloading, "42", "9"},   // 12.6/30; (30-12.6)/2 = 8.7 → 9
		{jobs.PhaseDownloading, "42", "nil"}, // stays at 42, speed unknown
		{jobs.PhaseFinishing, "100", "nil"},
	}
	if fmt.Sprint(*reports) != fmt.Sprint(want) {
		t.Fatalf("reports = %v, want %v", *reports, want)
	}
}

func TestProgressTrackerClampsAndNeedsALength(t *testing.T) {
	tracker, reports := recordingTracker(10)
	tracker.Line("size=1KiB time=00:00:12.00 bitrate=1kbits/s speed=1x")
	if want := []progressReport{{jobs.PhaseDownloading, "100", "0"}}; fmt.Sprint(*reports) != fmt.Sprint(want) {
		t.Fatalf("reports = %v, want %v", *reports, want)
	}

	tracker, reports = recordingTracker(0)
	tracker.Line("size=1KiB time=00:00:02.00 bitrate=1kbits/s speed=1x")
	if len(*reports) != 0 {
		t.Fatalf("expected no progress without a clip length, got %v", *reports)
	}
}

func TestLineWriterSplitsOnCarriageReturnsAndNewlines(t *testing.T) {
	var lines []string
	w := &lineWriter{fn: func(s string) { lines = append(lines, s) }}
	for _, chunk := range []string{"[download] Desti", "nation: x\nframe=1 time=00:00:01.00\rframe=2 ", "time=00:00:02.00\r\n\npartial"} {
		if n, err := w.Write([]byte(chunk)); err != nil || n != len(chunk) {
			t.Fatalf("Write = %d, %v", n, err)
		}
	}
	want := []string{"[download] Destination: x", "frame=1 time=00:00:01.00", "frame=2 time=00:00:02.00"}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Fatalf("lines = %q, want %q", lines, want)
	}
}

func TestLineWriterDropsOverlongLines(t *testing.T) {
	var lines []string
	w := &lineWriter{fn: func(s string) { lines = append(lines, s) }}
	_, _ = w.Write([]byte(strings.Repeat("x", maxLineLength+10) + "\nok\n"))
	if len(lines) != 2 || len(lines[0]) != maxLineLength || lines[1] != "ok" {
		t.Fatalf("unexpected lines: %d, first %d bytes", len(lines), len(lines[0]))
	}
}
