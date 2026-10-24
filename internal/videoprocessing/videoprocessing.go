package videoprocessing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MorrisMorrison/ytclipper-go/internal/config"
	"github.com/MorrisMorrison/ytclipper-go/internal/jobs"
)

var execContext = exec.CommandContext // allows mocking in tests

// waitDelay bounds how long a command's output pipes may stay open after the
// command was killed or exited (a variable for tests).
var waitDelay = 5 * time.Second

const (
	PresetVideo = "video"
	PresetAudio = "audio"

	// videoPresetFormat prefers H.264/AAC up to 1080p (plays everywhere) and
	// falls back to whatever is best up to 1080p.
	videoPresetFormat = "bv*[vcodec^=avc1][height<=1080]+ba[acodec^=mp4a]/b[vcodec^=avc1][height<=1080]/bv*[height<=1080]+ba/b"
	// audioPresetFormat is the best audio-only format, or the best combined
	// one when a video has no separate audio.
	audioPresetFormat = "ba/b"
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

// formatArgs returns the yt-dlp arguments that select what to download. info
// may be nil when the video info could not be loaded.
func formatArgs(req ClipRequest, info *VideoInfo) []string {
	switch req.Preset {
	case PresetVideo:
		return []string{"-f", videoPresetFormat, "--merge-output-format", "mp4"}
	case PresetAudio:
		return []string{"-f", audioPresetFormat, "-x", "--audio-format", "mp3", "--audio-quality", "192K"}
	}
	return []string{"-f", resolveFormat(req.Format, info)}
}

// resolveFormat merges best audio into a single video-only format, so a
// format picked from the list never yields a silent clip. MP4 video prefers
// M4A audio, which can be merged into MP4 without re-encoding.
func resolveFormat(format string, info *VideoInfo) string {
	if strings.Contains(format, "+") || info == nil {
		return format
	}
	f, ok := info.Format(format)
	if !ok || f.Kind != FormatKindVideo {
		return format
	}
	if f.Ext == "mp4" {
		return format + "+bestaudio[ext=m4a]/" + format + "+bestaudio"
	}
	return format + "+bestaudio"
}

// DownloadAndCutVideo downloads the section from-to of url. fileSizeLimit is
// enforced by ffmpeg (-fs), which yt-dlp always uses for sections and which
// ignores --max-filesize; ffmpeg stops writing at the limit, so callers check
// the size of the result. It returns stdout and stderr combined.
func DownloadAndCutVideo(outputTemplate string, formatArgs []string, fileSizeLimit int64, from string, to string, url string) ([]byte, error) {
	cmdArgs := []string{"-o", outputTemplate}
	cmdArgs = append(cmdArgs, formatArgs...)
	cmdArgs = append(cmdArgs,
		"--max-filesize", strconv.FormatInt(fileSizeLimit, 10),
		"--downloader-args", "ffmpeg:-fs "+strconv.FormatInt(fileSizeLimit, 10),
		"--download-sections", fmt.Sprintf("*%s-%s", from, to),
		url,
	)

	timeout := time.Duration(config.CONFIG.YtDlpConfig.DownloadTimeoutInSeconds) * time.Second
	out, err := execute(timeout, "yt-dlp", cmdArgs)
	return out.combined(), err
}

func ProcessClip(jobID string, req ClipRequest) {
	slots := jobSlots()
	slots <- struct{}{}
	defer func() { <-slots }()

	if !jobs.StartJob(jobID) {
		slog.Info("Job is no longer queued, skipping it", "job", jobID)
		return
	}
	slog.Info("Starting job", "job", jobID)

	// Cached (or shared with a concurrent lookup) in the common case: the page
	// loaded the info before asking for the clip.
	info, err := GetVideoInfo(req.VideoID)
	if err != nil {
		slog.Warn("Could not load video info, continuing without it", "job", jobID, "video", req.VideoID, "err", err)
		info = nil
	}
	if msg := checkAgainstDuration(req, info); msg != "" {
		jobs.FailJob(jobID, msg)
		return
	}

	clipDir := config.CONFIG.ClipCleanUpSchedulerConfig.ClipDirectoryPath
	if err := os.MkdirAll(clipDir, 0o755); err != nil {
		slog.Error("Could not create clip directory", "job", jobID, "dir", clipDir, "err", err)
		jobs.FailJob(jobID, "Could not store the clip")
		return
	}

	// yt-dlp picks the extension; the job ID keeps the name unique.
	sizeLimit := config.CONFIG.YtDlpConfig.ClipSizeInMb
	outputTemplate := filepath.Join(clipDir, jobID+".%(ext)s")
	output, err := DownloadAndCutVideo(outputTemplate, formatArgs(req, info), sizeLimit, req.From, req.To, WatchURL(req.VideoID))
	if err != nil {
		slog.Error("Clip download failed", "job", jobID, "err", err, "output", string(output))
		jobs.FailJob(jobID, summarizeError(output, err))
		return
	}

	outputPath, err := findOutputFile(clipDir, jobID)
	if err != nil {
		slog.Error("yt-dlp finished but no clip was written", "job", jobID, "err", err, "output", string(output))
		jobs.FailJob(jobID, "The clip could not be created")
		return
	}

	if st, err := os.Stat(outputPath); err == nil && sizeLimit > 0 && st.Size() >= sizeLimit {
		slog.Info("Clip reached the size limit, discarding it", "job", jobID, "bytes", st.Size())
		if err := os.Remove(outputPath); err != nil {
			slog.Error("Could not remove clip", "job", jobID, "path", outputPath, "err", err)
		}
		jobs.FailJob(jobID, sizeLimitMessage(sizeLimit))
		return
	}

	title := ""
	if info != nil {
		title = info.Title
	}
	fileName := DownloadName(title, req.VideoID, req.From, req.To, filepath.Ext(outputPath))

	slog.Info("Job completed", "job", jobID)
	jobs.CompleteJob(jobID, outputPath, fileName)
}

// checkAgainstDuration returns an error message when the clip ends after the
// video does. Without info, or for videos without a known duration (live
// streams), it accepts the request.
func checkAgainstDuration(req ClipRequest, info *VideoInfo) string {
	if info == nil || info.Duration <= 0 {
		return ""
	}
	to, err := ToSeconds(req.To)
	if err != nil || to <= info.Duration {
		return ""
	}
	return fmt.Sprintf("The end time is after the end of the video (%s).", formatDuration(info.Duration))
}

// formatDuration formats seconds as "m:ss" or, from an hour on, "h:mm:ss".
func formatDuration(seconds int) string {
	if seconds >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
	}
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

func sizeLimitMessage(limit int64) string {
	return fmt.Sprintf("The clip is larger than the %d MB limit. Choose a shorter range or a smaller format.", limit/(1024*1024))
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

// cmdOutput is what a command wrote, stdout and stderr kept apart so that
// machine-readable stdout (-J) is never mixed with log lines.
type cmdOutput struct {
	Stdout []byte
	Stderr []byte
}

// combined returns stdout followed by stderr, for logs and summarize (yt-dlp
// writes its ERROR lines to stderr).
func (o cmdOutput) combined() []byte {
	out := make([]byte, 0, len(o.Stdout)+len(o.Stderr)+1)
	out = append(out, o.Stdout...)
	if len(o.Stdout) > 0 && len(o.Stderr) > 0 && o.Stdout[len(o.Stdout)-1] != '\n' {
		out = append(out, '\n')
	}
	return append(out, o.Stderr...)
}

// execute runs a yt-dlp command with the common args and the given timeout.
func execute(timeout time.Duration, name string, baseArgs []string) (cmdOutput, error) {
	args := append(commonArgs(), baseArgs...)
	return executeWithTimeout(timeout, name, args...)
}

var errTimeout = errors.New("command timed out")

// executeWithTimeout runs a command and kills it when the timeout expires. On
// Unix the command gets its own process group and the whole group is killed,
// so children (the ffmpeg yt-dlp starts for --download-sections) die with it.
// waitDelay keeps a surviving child that holds the output pipes from blocking
// the caller.
func executeWithTimeout(timeout time.Duration, name string, args ...string) (cmdOutput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := execContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	cmd.WaitDelay = waitDelay
	killProcessGroupOnCancel(cmd)

	err := cmd.Run()
	out := cmdOutput{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if ctx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("%w after %v", errTimeout, timeout)
	}

	return out, err
}
