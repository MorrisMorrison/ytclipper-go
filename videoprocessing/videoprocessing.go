package videoprocessing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"ytclipper-go/config"
	"ytclipper-go/jobs"

	"github.com/MorrisMorrison/gutils/glogger"
)

var execContext = exec.CommandContext // allows mocking in tests

const (
	PresetVideo = "video"
	PresetAudio = "audio"

	// videoPresetFormat prefers H.264/AAC up to 1080p (plays everywhere) and
	// falls back to whatever is best up to 1080p.
	videoPresetFormat = "bv*[vcodec^=avc1][height<=1080]+ba[acodec^=mp4a]/b[vcodec^=avc1][height<=1080]/bv*[height<=1080]+ba/b"
)

// ClipRequest describes one clip job. Exactly one of Preset and Format is set;
// the caller has validated all fields.
type ClipRequest struct {
	VideoID string
	From    string // HH:MM:SS
	To      string // HH:MM:SS
	Preset  string // PresetVideo or PresetAudio
	Format  string // yt-dlp format id, "137" or "137+140"
}

// jobSlots limits how many clips are processed at once; further jobs wait in
// the queued state.
var jobSlots = sync.OnceValue(func() chan struct{} {
	return make(chan struct{}, config.CONFIG.MaxConcurrentJobs)
})

// formatArgs returns the yt-dlp arguments that select what to download.
func formatArgs(req ClipRequest) []string {
	switch req.Preset {
	case PresetVideo:
		return []string{"-f", videoPresetFormat, "--merge-output-format", "mp4"}
	case PresetAudio:
		return []string{"-f", "ba", "-x", "--audio-format", "mp3", "--audio-quality", "192K"}
	}
	return []string{"-f", resolveFormat(req.VideoID, req.Format)}
}

// resolveFormat merges best audio into a single video-only format, so a
// format picked from the list never yields a silent clip.
func resolveFormat(videoID string, format string) string {
	if strings.Contains(format, "+") {
		return format
	}
	info, err := GetVideoInfo(videoID)
	if err != nil {
		glogger.Log.Errorf(err, "Resolve Format: Could not load info for %s, using format %s as is", videoID, format)
		return format
	}
	if f, ok := info.Format(format); ok && f.Kind == FormatKindVideo {
		return format + "+bestaudio"
	}
	return format
}

func DownloadAndCutVideo(outputTemplate string, formatArgs []string, fileSizeLimit int64, from string, to string, url string) ([]byte, error) {
	cmdArgs := []string{"-o", outputTemplate}
	cmdArgs = append(cmdArgs, formatArgs...)
	cmdArgs = append(cmdArgs,
		"--max-filesize", fmt.Sprintf("%d", fileSizeLimit),
		"--download-sections", fmt.Sprintf("*%s-%s", from, to),
		url,
	)

	return execute("yt-dlp", cmdArgs)
}

func ProcessClip(jobID string, req ClipRequest) {
	slots := jobSlots()
	slots <- struct{}{}
	defer func() { <-slots }()

	glogger.Log.Infof("Process Clip: Start Job %s", jobID)
	jobs.StartJob(jobID)

	clipDir := config.CONFIG.ClipCleanUpSchedulerConfig.ClipDirectoryPath
	if err := os.MkdirAll(clipDir, 0o755); err != nil {
		glogger.Log.Errorf(err, "Process Clip: Could not create clip directory %s", clipDir)
		jobs.FailJob(jobID, "Could not store the clip")
		return
	}

	// yt-dlp picks the extension; the job ID keeps the name unique.
	outputTemplate := filepath.Join(clipDir, jobID+".%(ext)s")
	output, err := DownloadAndCutVideo(outputTemplate, formatArgs(req), config.CONFIG.YtDlpConfig.ClipSizeInMb, req.From, req.To, WatchURL(req.VideoID))
	if err != nil {
		glogger.Log.Errorf(err, "Process Clip: Failed to download video. Output:\n%s", string(output))
		jobs.FailJob(jobID, summarizeError(output, err))
		return
	}

	outputPath, err := findOutputFile(clipDir, jobID)
	if err != nil {
		glogger.Log.Errorf(err, "Process Clip: yt-dlp finished but no clip was written. Output:\n%s", string(output))
		jobs.FailJob(jobID, "The clip could not be created (it may exceed the size limit)")
		return
	}

	title := ""
	if info, ok := CachedVideoInfo(req.VideoID); ok {
		title = info.Title
	}
	fileName := DownloadName(title, req.VideoID, req.From, req.To, filepath.Ext(outputPath))

	glogger.Log.Infof("Process Clip: Complete Job %s", jobID)
	jobs.CompleteJob(jobID, outputPath, fileName)
}

// findOutputFile returns the finished file yt-dlp wrote for a job, ignoring
// partial downloads.
func findOutputFile(dir string, jobID string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, jobID+".*"))
	if err != nil {
		return "", err
	}
	for _, m := range matches {
		if !strings.HasSuffix(m, ".part") && !strings.HasSuffix(m, ".ytdl") {
			return m, nil
		}
	}
	return "", fmt.Errorf("no output file for job %s", jobID)
}

// summarizeError turns yt-dlp output of a failed download into a short
// message that is safe to show to users.
func summarizeError(output []byte, err error) string {
	return summarize(output, err, "Timed out while downloading the clip", "Failed to download the clip")
}

// summarize returns the last yt-dlp ERROR line, without the rest of the
// output (which can contain the command line), or a fallback message.
func summarize(output []byte, err error, timeoutMsg string, fallbackMsg string) string {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "ERROR:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "ERROR:"))
		}
	}
	if errors.Is(err, errTimeout) {
		return timeoutMsg
	}
	return fallbackMsg
}

// commonArgs returns the flags applied to every yt-dlp invocation: the optional
// proxy and yt-dlp's own retry/quiet flags.
func commonArgs() []string {
	var args []string

	if config.CONFIG.YtDlpConfig.Proxy != "" {
		args = append(args, "--proxy", config.CONFIG.YtDlpConfig.Proxy)
	}

	if config.CONFIG.YtDlpConfig.ExtractorRetries > 0 {
		args = append(args, "--extractor-retries", strconv.Itoa(config.CONFIG.YtDlpConfig.ExtractorRetries))
	}

	args = append(args, "--no-warnings")

	return args
}

func execute(name string, baseArgs []string) ([]byte, error) {
	timeout := time.Duration(config.CONFIG.YtDlpConfig.CommandTimeoutInSeconds) * time.Second
	args := append(commonArgs(), baseArgs...)
	return executeWithTimeout(timeout, name, args...)
}

var errTimeout = errors.New("command timed out")

func executeWithTimeout(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := execContext(ctx, name, args...)

	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("%w after %v", errTimeout, timeout)
	}

	return output, err
}
