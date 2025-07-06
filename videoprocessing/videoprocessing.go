package videoprocessing

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"ytclipper-go/config"
	"ytclipper-go/jobs"

	"github.com/MorrisMorrison/gutils/glogger"
)

const videoOutputDir = "./videos"
const throttledStatus = "THROTTLED"

var execContext = exec.CommandContext // allows mocking in tests

func DownloadAndCutVideo(outputPath string, selectedFormat string, fileSizeLimit int64, from string, to string, url string) ([]byte, error) {
	cmdArgs := []string{
		"-o", outputPath,
		"-f", selectedFormat,
		"-v",
		"--max-filesize", fmt.Sprintf("%d", fileSizeLimit),
		"--downloader", "ffmpeg",
		"--downloader-args", fmt.Sprintf("ffmpeg_i:-ss %s -to %s", from, to),
		url,
	}

	return executeWithFallback("yt-dlp", cmdArgs)
}

func ProcessClip(jobID string, url string, from string, to string, selectedFormat string) {
	glogger.Log.Infof("Process Clip: Start Job %s", jobID)
	jobs.StartJob(jobID)

	availableFormats, err := GetAvailableFormats(url)
	if err != nil {
		glogger.Log.Error(err, "Process Clip: Failed to retrieve formats")
		jobs.FailJob(jobID, fmt.Sprintf("Failed to retrieve formats: %v", err))
		return
	}

	fileExtension, err := getFileExtensionFromFormatID(selectedFormat, availableFormats)
	if err != nil {
		glogger.Log.Errorf(err, "Process Clip: Unsupported format ID: %s", selectedFormat)
		jobs.FailJob(jobID, fmt.Sprintf("Unsupported format ID: %s", selectedFormat))
		return
	}

	outputPath := filepath.Join(videoOutputDir, fmt.Sprintf("%s%s", filepath.Base(jobID), fileExtension))
	output, err := DownloadAndCutVideo(outputPath, selectedFormat, config.CONFIG.YtDlpConfig.ClipSizeInMb, from, to, url)
	if err != nil {
		glogger.Log.Errorf(err, "Process Clip: Failed to download video: %s", string(output))
		jobs.FailJob(jobID, fmt.Sprintf("Failed to download video: %s", string(output)))
		return
	}

	glogger.Log.Infof("Process Clip: Complete Job %s", jobID)
	jobs.CompleteJob(jobID, outputPath)
}

func GetAvailableFormats(url string) ([]map[string]string, error) {
	glogger.Log.Infof("Get Available Formats: Fetching available formats for URL %s", url)

	cmdArgs := []string{"-F", url}

	output, err := executeWithFallback("yt-dlp", cmdArgs)
	if err != nil {
		glogger.Log.Errorf(err, "Get Available Formats: Error executing yt-dlp. Output\n%s", string(output))
		return nil, fmt.Errorf("yt-dlp failed: %w", err)
	}

	if config.CONFIG.Debug {
		glogger.Log.Infof("Get Available Formats: yt-dlp command succeeded. Output:\n%s", output)
	}

	formats := parseFormats(string(output))
	if formats == nil {
		glogger.Log.Errorf(fmt.Errorf("Formats are nil"), "Get Video Duration: Could not find any available formats. Output:/n%s", output)
	}

	return parseFormats(string(output)), nil
}

func GetVideoDuration(url string) (string, error) {
	glogger.Log.Infof("Get Video Duration: Fetch duration for URL %s", url)

	cmdArgs := []string{
		"--get-duration",
		"--no-warnings",
		url,
	}

	output, err := executeWithFallback("yt-dlp", cmdArgs)
	if err != nil {
		glogger.Log.Errorf(err, "Get Video Duration: Error executing yt-dlp. Output\n%s", string(output))
		return "", err
	}

	if config.CONFIG.Debug {
		glogger.Log.Infof("Get Video Duration: yt-dlp command succeeded. Output:\n%s", output)
	}

	duration := ExtractDuration(string(output))
	return duration, nil
}

func ExtractDuration(output string) string {
	reWithColon := regexp.MustCompile(`\d+:\d{2}(?::\d{2})?`)
	matches := reWithColon.FindString(output)
	if matches != "" {
		return matches
	}

	reJustSeconds := regexp.MustCompile(`(?m)^\s*(\d+)\s*$`)
	secMatches := reJustSeconds.FindStringSubmatch(output)
	if len(secMatches) > 1 {
		return secMatches[1]
	}

	return ""
}

func parseFormats(output string) []map[string]string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		glogger.Log.Warning("Empty output received when parsing formats")
		return nil
	}

	formatRegex := regexp.MustCompile(`(?m)^\s*(\d+)\s+(\w+)\s+(audio only|video only|\d+x\d+|\d+p)\s+([a-zA-Z0-9\.\-\_]+)?\s*([\~\d\.]+(?:[kKM]i?B)?)?.*?\|\s+(.*?)$`)
	formats := make([]map[string]string, 0, len(lines))

	for _, line := range lines {
		if format := parseFormatLine(line, formatRegex); format != nil {
			formats = append(formats, format)
		}
	}

	return formats
}

func parseFormatLine(line string, formatRegex *regexp.Regexp) map[string]string {
	matches := formatRegex.FindStringSubmatch(line)
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
	bitrateRegex := regexp.MustCompile(`([\~\d\.]+[kKM]i?B)`)
	match := bitrateRegex.FindStringSubmatch(additional)
	if len(match) > 0 {
		return match[1]
	}

	glogger.Log.Warningf("Extract Bitrate: Could not extract bitrate from %s", additional)
	return "N/A"
}

func getFileExtensionFromFormatID(formatID string, formats []map[string]string) (string, error) {
	for _, format := range formats {
		if format["id"] == formatID {
			if ext, exists := format["extension"]; exists {
				return fmt.Sprintf(".%s", ext), nil
			}
		}
	}
	return "", fmt.Errorf("format ID not found")
}

func applyAntiDetectionArgs(cmdArgs []string) []string {
	var args []string

	// Apply proxy if configured
	if config.CONFIG.YtDlpConfig.Proxy != "" {
		glogger.Log.Infof("Using proxy: %s", config.CONFIG.YtDlpConfig.Proxy)
		args = append(args, "--proxy", config.CONFIG.YtDlpConfig.Proxy)
	}

	// Apply cookies file if configured
	if config.CONFIG.YtDlpConfig.CookiesFile != "" {
		glogger.Log.Infof("Using cookies file: %s", config.CONFIG.YtDlpConfig.CookiesFile)
		args = append(args, "--cookies", config.CONFIG.YtDlpConfig.CookiesFile)
	}

	// Apply user agent
	if config.CONFIG.YtDlpConfig.UserAgent != "" {
		args = append(args, "--user-agent", config.CONFIG.YtDlpConfig.UserAgent)
	}

	// Apply extractor retries
	if config.CONFIG.YtDlpConfig.ExtractorRetries > 0 {
		args = append(args, "--extractor-retries", fmt.Sprintf("%d", config.CONFIG.YtDlpConfig.ExtractorRetries))
	}

	// Add anti-bot detection arguments
	args = append(args,
		"--sleep-requests", "1", // Sleep between requests
		"--sleep-interval", "1", // Random sleep interval
		"--max-sleep-interval", "3", // Maximum sleep interval
	)

	return append(args, cmdArgs...)
}

func executeWithFallback(name string, baseArgs []string) ([]byte, error) {
	timeout := time.Duration(config.CONFIG.YtDlpConfig.CommandTimeoutInSeconds) * time.Second

	// First attempt: with full anti-detection (including proxy)
	args := applyAntiDetectionArgs(baseArgs)
	glogger.Log.Infof("Attempting yt-dlp with full anti-detection configuration")
	output, err := executeWithTimeout(timeout, name, args...)

	// Check if proxy-related error
	if err != nil && config.CONFIG.YtDlpConfig.Proxy != "" && (strings.Contains(string(output), "proxy") ||
		strings.Contains(string(output), "Tunnel connection failed") ||
		strings.Contains(string(output), "Unable to connect to proxy")) {

		glogger.Log.Warningf("Proxy connection failed, attempting without proxy: %v", err)

		// Second attempt: without proxy but with other anti-detection measures
		args = applyAntiDetectionArgsWithoutProxy(baseArgs)
		output, err = executeWithTimeout(timeout, name, args...)

		if err != nil {
			glogger.Log.Warningf("Anti-detection without proxy failed, attempting basic execution: %v", err)

			// Third attempt: basic execution with minimal arguments
			output, err = executeWithTimeout(timeout, name, baseArgs...)
		}
	}

	return output, err
}

func applyAntiDetectionArgsWithoutProxy(cmdArgs []string) []string {
	var args []string

	if config.CONFIG.YtDlpConfig.CookiesFile != "" {
		glogger.Log.Infof("Using cookies file: %s", config.CONFIG.YtDlpConfig.CookiesFile)
		args = append(args, "--cookies", config.CONFIG.YtDlpConfig.CookiesFile)
	}

	if config.CONFIG.YtDlpConfig.UserAgent != "" {
		args = append(args, "--user-agent", config.CONFIG.YtDlpConfig.UserAgent)
	}

	if config.CONFIG.YtDlpConfig.ExtractorRetries > 0 {
		args = append(args, "--extractor-retries", fmt.Sprintf("%d", config.CONFIG.YtDlpConfig.ExtractorRetries))
	}

	args = append(args,
		"--sleep-requests", "1",
		"--sleep-interval", "1",
		"--max-sleep-interval", "3",
	)

	return append(args, cmdArgs...)
}

func executeWithTimeout(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := execContext(ctx, name, args...)

	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("command timed out after %v", timeout)
	}

	return output, err
}
