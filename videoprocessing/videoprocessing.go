package videoprocessing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"ytclipper-go/config"
	"ytclipper-go/jobs"

	"github.com/MorrisMorrison/gutils/glogger"
)

const throttledStatus = "THROTTLED"

var execContext = exec.CommandContext // allows mocking in tests

var (
	durationWithColonRegex = regexp.MustCompile(`\d+:\d{2}(?::\d{2})?`)
	durationSecondsRegex   = regexp.MustCompile(`(?m)^\s*(\d+)\s*$`)
	formatLineRegex        = regexp.MustCompile(`(?m)^\s*(\d+)\s+(\w+)\s+(audio only|video only|\d+x\d+|\d+p)\s+([a-zA-Z0-9\.\-\_]+)?\s*([\~\d\.]+(?:[kKM]i?B)?)?.*?\|\s+(.*?)$`)
	bitrateRegex           = regexp.MustCompile(`([\~\d\.]+[kKM]i?B)`)
)

// jobSlots limits how many clips are processed at once; further jobs wait in
// the queued state.
var jobSlots = sync.OnceValue(func() chan struct{} {
	return make(chan struct{}, config.CONFIG.MaxConcurrentJobs)
})

func DownloadAndCutVideo(outputTemplate string, selectedFormat string, fileSizeLimit int64, from string, to string, url string) ([]byte, error) {
	cmdArgs := []string{
		"-o", outputTemplate,
		"-f", selectedFormat,
		"--max-filesize", fmt.Sprintf("%d", fileSizeLimit),
		"--download-sections", fmt.Sprintf("*%s-%s", from, to),
		url,
	}

	return execute("yt-dlp", cmdArgs)
}

func ProcessClip(jobID string, url string, from string, to string, selectedFormat string) {
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
	output, err := DownloadAndCutVideo(outputTemplate, selectedFormat, config.CONFIG.YtDlpConfig.ClipSizeInMb, from, to, url)
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

	glogger.Log.Infof("Process Clip: Complete Job %s", jobID)
	jobs.CompleteJob(jobID, outputPath)
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

// summarizeError turns yt-dlp output into a short message that is safe to show
// to users: the last ERROR line, without the full command output.
func summarizeError(output []byte, err error) string {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "ERROR:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "ERROR:"))
		}
	}
	if errors.Is(err, errTimeout) {
		return "Timed out while downloading the clip"
	}
	return "Failed to download the clip"
}

func GetAvailableFormats(url string) ([]map[string]string, error) {
	glogger.Log.Infof("Get Available Formats: Fetching available formats for URL %s", url)

	output, err := execute("yt-dlp", []string{"-F", url})
	if err != nil {
		glogger.Log.Errorf(err, "Get Available Formats: Error executing yt-dlp. Output\n%s", string(output))
		return nil, fmt.Errorf("yt-dlp failed: %w", err)
	}

	if config.CONFIG.Debug {
		glogger.Log.Infof("Get Available Formats: yt-dlp command succeeded. Output:\n%s", output)
	}

	formats := parseFormats(string(output))
	if len(formats) == 0 {
		glogger.Log.Errorf(errors.New("no formats"), "Get Available Formats: Could not find any available formats. Output:\n%s", output)
		return nil, errors.New("could not find any available formats")
	}

	return formats, nil
}

func GetVideoDuration(url string) (string, error) {
	glogger.Log.Infof("Get Video Duration: Fetch duration for URL %s", url)

	output, err := execute("yt-dlp", []string{"--get-duration", url})
	if err != nil {
		glogger.Log.Errorf(err, "Get Video Duration: Error executing yt-dlp. Output\n%s", string(output))
		return "", err
	}

	if config.CONFIG.Debug {
		glogger.Log.Infof("Get Video Duration: yt-dlp command succeeded. Output:\n%s", output)
	}

	return ExtractDuration(string(output)), nil
}

func ExtractDuration(output string) string {
	if match := durationWithColonRegex.FindString(output); match != "" {
		return match
	}

	if secMatches := durationSecondsRegex.FindStringSubmatch(output); len(secMatches) > 1 {
		return secMatches[1]
	}

	return ""
}

func parseFormats(output string) []map[string]string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	formats := make([]map[string]string, 0, len(lines))

	for _, line := range lines {
		if format := parseFormatLine(line); format != nil {
			formats = append(formats, format)
		}
	}

	return formats
}

func parseFormatLine(line string) map[string]string {
	matches := formatLineRegex.FindStringSubmatch(line)
	if len(matches) == 0 {
		return nil
	}

	additional := strings.TrimSpace(matches[6])
	if strings.Contains(additional, throttledStatus) {
		return nil
	}

	return map[string]string{
		"id":         matches[1],
		"extension":  matches[2],
		"label":      matches[3],
		"codec":      strings.TrimSpace(matches[4]),
		"bitrate":    extractBitrate(matches[6]),
		"formatType": determineFormatType(matches[3]),
		"additional": additional,
	}
}

func determineFormatType(label string) string {
	switch label {
	case "audio only":
		return "audio only"
	case "video only":
		return "video only"
	default:
		return "audio and video"
	}
}

func extractBitrate(additional string) string {
	if match := bitrateRegex.FindStringSubmatch(additional); len(match) > 0 {
		return match[1]
	}
	return "N/A"
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
