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

	cmdArgs = applyProxyArgs(cmdArgs)

	output, err := executeWithTimeout(time.Duration(config.CONFIG.YtDlpConfig.CommandTimeoutInSeconds)*time.Second, "yt-dlp", cmdArgs...)
	return output, err
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
	cmdArgs = applyProxyArgs(cmdArgs)

    output, err := executeWithTimeout(time.Duration(config.CONFIG.YtDlpConfig.CommandTimeoutInSeconds)*time.Second, "yt-dlp", cmdArgs...)
    if err != nil {
        glogger.Log.Errorf(err, "Get Available Formats: Error executing yt-dlp. Output\n%s", string(output))
        return nil, fmt.Errorf("yt-dlp failed: %w", err)
    }

    if (config.CONFIG.Debug){
        glogger.Log.Infof("Get Available Formats: yt-dlp command succeeded. Output:\n%s", output) 
    }

    formats:=parseFormats(string(output))
    if (formats == nil){
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

	cmdArgs = applyProxyArgs(cmdArgs)
	output, err := executeWithTimeout(time.Duration(config.CONFIG.YtDlpConfig.CommandTimeoutInSeconds)*time.Second, "yt-dlp", cmdArgs...)
	if err != nil {
        glogger.Log.Errorf(err, "Get Video Duration: Error executing yt-dlp. Output\n%s", string(output))
		return "", err
	}

    if (config.CONFIG.Debug){
        glogger.Log.Infof("Get Video Duration: yt-dlp command succeeded. Output:\n%s", output) 
    }

	duration := ExtractDuration(string(output))
	return duration, nil
}

func ExtractDuration(output string) string {
    re := regexp.MustCompile(`\d+:\d{2}(?::\d{2})?`)
    matches := re.FindStringSubmatch(output)
    if len(matches) > 0 {
        return matches[0]
    }
    return ""
}

func parseFormats(output string) []map[string]string {
    lines := strings.Split(output, "\n")

    formatRegex := regexp.MustCompile(`(?m)^\s*(\d+)\s+(\w+)\s+(audio only|video only|\d+x\d+|\d+p)\s+([a-zA-Z0-9\.\-\_]+)?\s*([\~\d\.]+(?:[kKM]i?B)?)?.*?\|\s+(.*?)$`)
    var formats []map[string]string

    for _, line := range lines {
        matches := formatRegex.FindStringSubmatch(line)
        if len(matches) > 0 {
            additional := strings.TrimSpace(matches[6])
            if strings.Contains(additional, "THROTTLED") {
                continue
            }

            bitrate := extractBitrate(matches[6])

            formatType := "audio and video" 
            if matches[3] == "audio only" {
                formatType = "audio only"
            } else if matches[3] == "video only" {
                formatType = "video only"
            }

            formats = append(formats, map[string]string{
                "id":         matches[1],                 
                "extension":  matches[2],                 
                "label":      matches[3],                
                "codec":      strings.TrimSpace(matches[4]), 
                "bitrate":    bitrate,                   
                "formatType": formatType,                
                "additional": additional, 
            })
        }
    }

    return formats
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

func applyProxyArgs(cmdArgs []string) []string {
	if config.CONFIG.YtDlpConfig.Proxy != "" {
        glogger.Log.Infof("Using proxy: %s", config.CONFIG.YtDlpConfig.Proxy)
		return append([]string{"--proxy", config.CONFIG.YtDlpConfig.Proxy}, cmdArgs...)
	}
	return cmdArgs
}

func executeWithTimeout(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := execContext(ctx, name, args...)

	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("command timed out")
	}

	return output, err
}
