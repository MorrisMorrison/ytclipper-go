package videoprocessing

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"ytclipper-go/config"
	"ytclipper-go/jobs"
)

const videoOutputDir = "./videos"
var execCommand = exec.Command 

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

	if config.CONFIG.YtDlp != "" {
        	log.Printf("Using proxy: %s", config.CONFIG.YtDlp)
        	cmdArgs = append([]string{"--proxy", config.CONFIG.YtDlpProxy}, cmdArgs...)
    	}

	cmd := execCommand("yt-dlp", cmdArgs...)
	output, err := cmd.CombinedOutput()

	return output, err
}

func ProcessClip(jobID string, url string, from string, to string, selectedFormat string) {
    jobs.StartJob(jobID)

    availableFormats, err := GetAvailableFormats(url)
    if err != nil {
        jobs.FailJob(jobID, fmt.Sprintf("Failed to retrieve formats: %v", err))
        return
    }

    fileExtension, err := getFileExtensionFromFormatID(selectedFormat, availableFormats)
    if err != nil {
        jobs.FailJob(jobID, fmt.Sprintf("Unsupported format ID: %s", selectedFormat))
        return
    }

    outputPath := filepath.Join(videoOutputDir, fmt.Sprintf("%s%s", filepath.Base(jobID), fileExtension))
    output, err := DownloadAndCutVideo(outputPath, selectedFormat, config.CONFIG.ClipSizeInMb, from, to, url)
    if err != nil {
        jobs.FailJob(jobID, fmt.Sprintf("Failed to download video: %s", string(output)))
        return
    }

    jobs.CompleteJob(jobID, outputPath)
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

func GetAvailableFormats(url string) ([]map[string]string, error) {
    log.Printf("Fetching available formats for URL: %s", url)

    cmdArgs := []string{"-F", url}

    if config.CONFIG.YtDlp != "" {
        log.Printf("Using proxy: %s", config.CONFIG.YtDlp)
        cmdArgs = append([]string{"--proxy", config.CONFIG.YtDlpProxy}, cmdArgs...)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    cmd := exec.CommandContext(ctx, "yt-dlp", cmdArgs...)

    output, err := cmd.CombinedOutput()
    if err != nil {
        log.Printf("Error executing yt-dlp: %v", err)
        log.Printf("yt-dlp output:\n%s", output)
        return nil, fmt.Errorf("yt-dlp failed: %w", err)
    }

    log.Printf("yt-dlp command succeeded. Output:\n%s", output)
    return parseFormats(string(output)), nil
}



func GetVideoDuration(url string) (string, error) {
	cmdArgs := []string{
		"--get-duration",
		"--no-warnings",
		url,
	}
	cmd := execCommand("yt-dlp", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
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
    return "N/A" 
}
