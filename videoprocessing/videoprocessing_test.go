package videoprocessing

import (
	"context"
	"fmt"
	"os/exec"
	"reflect"
	"testing"
)

func TestExtractDuration(t *testing.T) {
	output := "Duration: 00:01:30"
	expected := "00:01:30"

	duration := ExtractDuration(output)
	if duration != expected {
		t.Errorf("Expected %s, but got %s", expected, duration)
	}
}

func TestExtractBitrate(t *testing.T) {
	tests := []struct {
		name       string
		additional string
		expected   string
	}{
		{
			name:       "Valid bitrate in KiB",
			additional: "128KiB",
			expected:   "128KiB",
		},
		{
			name:       "Valid bitrate in MiB",
			additional: "1.2MiB",
			expected:   "1.2MiB",
		},
		{
			name:       "Valid bitrate with tilde (~)",
			additional: "~192KiB",
			expected:   "~192KiB",
		},
		{
			name:       "No bitrate",
			additional: "Some other text",
			expected:   "N/A",
		},
		{
			name:       "Empty string",
			additional: "",
			expected:   "N/A",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := extractBitrate(test.additional)
			if result != test.expected {
				t.Errorf("For input '%s', expected '%s', but got '%s'", test.additional, test.expected, result)
			}
		})
	}
}

func TestDownloadAndCutVideo(t *testing.T) {
	originalExecContext := execContext
	defer func() { execContext = originalExecContext }()

	var capturedArgs []string
	execContext = func(ctx context.Context,name string, arg ...string) *exec.Cmd {
		capturedArgs = append([]string{name}, arg...)
		capturedArgs = append([]string{name}, arg...)
		return exec.Command("echo", "mock") 
	}

	outputPath := "./videos/test_clip.mp4"
	selectedFormat := "22"
	fileSizeLimit := int64(500000000)
	from := "00:00:30"
	to := "00:01:00"
	url := "https://www.youtube.com/watch?v=example"

	_, _ = DownloadAndCutVideo(outputPath, selectedFormat, fileSizeLimit, from, to, url)

	expectedArgs := []string{
		"yt-dlp",
		"-o", outputPath,
		"-f", selectedFormat,
		"-v",
		"--max-filesize", fmt.Sprintf("%d", fileSizeLimit),
		"--downloader", "ffmpeg",
		"--downloader-args", fmt.Sprintf("ffmpeg_i:-ss %s -to %s", from, to),
		url,
	}

	if !reflect.DeepEqual(capturedArgs, expectedArgs) {
		t.Errorf("Expected command args: %v, but got: %v", expectedArgs, capturedArgs)
	}
}

func TestGetVideoDuration(t *testing.T) {
	originalExecContext := execContext
	defer func() { execContext = originalExecContext }()

	var capturedArgs []string
	execContext = func(ctx context.Context,name string, arg ...string) *exec.Cmd {
		capturedArgs = append([]string{name}, arg...)
		return exec.Command("echo", "00:03:45") 
	}

	url := "https://www.youtube.com/watch?v=example"

	duration, err := GetVideoDuration(url)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	expectedDuration := "00:03:45"
	if duration != expectedDuration {
		t.Errorf("Expected duration: %s, but got: %s", expectedDuration, duration)
	}

	expectedArgs := []string{
		"yt-dlp",
		"--get-duration",
		"--no-warnings",
		url,
	}

	if !reflect.DeepEqual(capturedArgs, expectedArgs) {
		t.Errorf("Expected command args: %v, but got: %v", expectedArgs, capturedArgs)
	}
}