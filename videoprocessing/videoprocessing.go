package videoprocessing

import (
	"context"
	"fmt"
	"math/rand"
	"os"
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
	return GetAvailableFormatsWithContext(url, "", "")
}

func GetAvailableFormatsWithContext(url, userAgent, cookies string) ([]map[string]string, error) {
	glogger.Log.Infof("Get Available Formats: Fetching available formats for URL %s", url)

	cmdArgs := []string{"-F", url}

	output, err := executeWithFallbackAndContext("yt-dlp", cmdArgs, userAgent, cookies)
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
	return GetVideoDurationWithContext(url, "", "")
}

func GetVideoDurationWithContext(url, userAgent, cookies string) (string, error) {
	glogger.Log.Infof("Get Video Duration: Fetch duration for URL %s", url)

	cmdArgs := []string{
		"--get-duration",
		"--no-warnings",
		url,
	}

	output, err := executeWithFallbackAndContext("yt-dlp", cmdArgs, userAgent, cookies)
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

// getUserAgent returns a rotating user agent or the configured one
func getUserAgent() string {
	if !config.CONFIG.YtDlpConfig.EnableUserAgentRotation {
		if config.CONFIG.YtDlpConfig.UserAgent != "" {
			return config.CONFIG.YtDlpConfig.UserAgent
		}
	}

	// Modern browser user agents (2024)
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:122.0) Gecko/20100101 Firefox/122.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2.1 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36 Edg/121.0.0.0",
	}

	return userAgents[rand.Intn(len(userAgents))]
}

// createTempCookieFile converts browser cookies to Netscape format and creates a temporary file
func createTempCookieFile(cookies string) (string, error) {
	glogger.Log.Infof("Creating temp cookie file with %d chars of cookies", len(cookies))

	// Create temporary file
	tempFile, err := os.CreateTemp("", "ytclipper-cookies-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	glogger.Log.Infof("Created temp cookie file: %s", tempFile.Name())

	// Write Netscape cookie file header
	header := "# Netscape HTTP Cookie File\n# This is a generated file! Do not edit.\n\n"
	if _, err := tempFile.WriteString(header); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write header: %w", err)
	}

	// Parse and convert cookies to Netscape format
	cookiePairs := strings.Split(cookies, ";")
	for _, pair := range cookiePairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Skip empty cookies
		if name == "" || value == "" {
			continue
		}

		// Convert to Netscape format: domain \t flag \t path \t secure \t expiration \t name \t value
		// We'll use YouTube domain and set reasonable defaults
		netscapeLine := fmt.Sprintf(".youtube.com\tTRUE\t/\tTRUE\t%d\t%s\t%s\n",
			time.Now().Add(365*24*time.Hour).Unix(), // Expire in 1 year
			name,
			value)

		if _, err := tempFile.WriteString(netscapeLine); err != nil {
			os.Remove(tempFile.Name())
			return "", fmt.Errorf("failed to write cookie: %w", err)
		}
	}

	// Schedule cleanup after 5 minutes
	go func() {
		time.Sleep(5 * time.Minute)
		os.Remove(tempFile.Name())
		glogger.Log.Infof("Cleaned up temporary cookie file: %s", tempFile.Name())
	}()

	return tempFile.Name(), nil
}

func applyAntiDetectionArgsNoCookiesProxy(cmdArgs []string) []string {
	var args []string

	// Use rotating user agent
	userAgent := getUserAgent()
	glogger.Log.Infof("Using user agent: %s", userAgent)
	args = append(args, "--user-agent", userAgent)

	// Apply extractor retries
	if config.CONFIG.YtDlpConfig.ExtractorRetries > 0 {
		args = append(args, "--extractor-retries", fmt.Sprintf("%d", config.CONFIG.YtDlpConfig.ExtractorRetries))
	}

	// Enhanced anti-bot detection arguments
	sleepInterval := config.CONFIG.YtDlpConfig.SleepInterval
	maxSleep := sleepInterval * 3
	if maxSleep < 5 {
		maxSleep = 5
	}

	args = append(args,
		"--sleep-requests", fmt.Sprintf("%d", sleepInterval),
		"--sleep-interval", fmt.Sprintf("%d", sleepInterval),
		"--max-sleep-interval", fmt.Sprintf("%d", maxSleep),
		"--no-check-certificate", // Skip SSL verification
		"--geo-bypass",           // Attempt to bypass geographic restrictions
		"--no-warnings",          // Reduce output noise
		"--prefer-free-formats",  // Prefer free formats
		"--no-abort-on-error",    // Continue on errors
		"--ignore-errors",        // Ignore download errors
		"--add-header", "Accept-Language:en-US,en;q=0.9",
		"--add-header", "Accept:text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"--add-header", "Accept-Encoding:gzip, deflate, br",
		"--add-header", "DNT:1",
		"--add-header", "Connection:keep-alive",
		"--add-header", "Upgrade-Insecure-Requests:1",
		"--add-header", "Sec-Fetch-Dest:document",
		"--add-header", "Sec-Fetch-Mode:navigate",
		"--add-header", "Sec-Fetch-Site:none",
		"--add-header", "Sec-Fetch-User:?1",
		"--add-header", "Cache-Control:max-age=0",
	)

	return append(args, cmdArgs...)
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
	} else {
		// Fallback to rotating user agent
		args = append(args, "--user-agent", getUserAgent())
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

	// Strategy 1: Enhanced anti-detection without cookies/proxy
	args := applyAntiDetectionArgsNoCookiesProxy(baseArgs)
	glogger.Log.Infof("Attempting yt-dlp with enhanced anti-detection (no cookies/proxy)")
	output, err := executeWithTimeout(timeout, name, args...)

	if err != nil {
		glogger.Log.Warningf("Enhanced anti-detection failed: %v", err)

		// Strategy 2: Try with legacy full configuration (includes cookies/proxy if available)
		legacyArgs := applyAntiDetectionArgs(baseArgs)
		glogger.Log.Infof("Attempting yt-dlp with legacy full anti-detection configuration")
		output, err = executeWithTimeout(timeout, name, legacyArgs...)
	}

	return output, err
}

func executeWithFallbackAndContext(name string, baseArgs []string, userAgent, cookies string) ([]byte, error) {
	timeout := time.Duration(config.CONFIG.YtDlpConfig.CommandTimeoutInSeconds) * time.Second

	// Strategy 1: Use user's browser context if available
	if userAgent != "" || cookies != "" {
		userArgs := applyUserContext(baseArgs, userAgent, cookies)
		glogger.Log.Infof("Attempting yt-dlp with user browser context")
		output, err := executeWithTimeout(timeout, name, userArgs...)
		if err == nil {
			return output, err
		}
		glogger.Log.Warningf("User context failed: %v", err)
	}

	// Strategy 2: Fallback to legacy full configuration (includes cookies/proxy if available)
	legacyArgs := applyAntiDetectionArgs(baseArgs)
	glogger.Log.Infof("Attempting yt-dlp with legacy full anti-detection configuration")
	return executeWithTimeout(timeout, name, legacyArgs...)
}

func applyUserContext(cmdArgs []string, userAgent, cookies string) []string {
	var args []string

	// Use provided user agent if available
	if userAgent != "" {
		glogger.Log.Infof("Using user's browser user agent")
		args = append(args, "--user-agent", userAgent)
	} else {
		// Fallback to rotating user agent
		args = append(args, "--user-agent", getUserAgent())
	}

	// Try different cookie approaches
	cookieApplied := false
	
	// First try: Use provided cookies if available
	if cookies != "" {
		glogger.Log.Infof("Using user's browser cookies")
		// Create temporary cookies file in Netscape format
		if cookieFile, err := createTempCookieFile(cookies); err == nil {
			args = append(args, "--cookies", cookieFile)
			cookieApplied = true
		} else {
			glogger.Log.Warningf("Failed to create cookie file: %v", err)
		}
	}
	
	// Second try: Use browser cookie extraction if no manual cookies provided
	if !cookieApplied {
		glogger.Log.Infof("Attempting to extract cookies from browser")
		// Try common browsers in order of likelihood
		browsers := []string{"chrome", "firefox", "safari", "edge"}
		for _, browser := range browsers {
			args = append(args, "--cookies-from-browser", browser)
			glogger.Log.Infof("Added --cookies-from-browser %s", browser)
			break // Only try the first one for now
		}
	}

	// Apply minimal anti-detection
	args = append(args,
		"--extractor-retries", "3",
		"--sleep-requests", "1",
		"--sleep-interval", "1",
		"--max-sleep-interval", "3",
		"--no-warnings",
		"--add-header", "Accept-Language:en-US,en;q=0.9",
		"--add-header", "Accept:text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"--add-header", "Accept-Encoding:gzip, deflate, br",
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
