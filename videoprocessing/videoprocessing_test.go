package videoprocessing

import (
	"context"
	"os/exec"
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

	var allCapturedArgs [][]string
	execContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		cmdArgs := append([]string{name}, arg...)
		allCapturedArgs = append(allCapturedArgs, cmdArgs)
		return exec.Command("echo", "mock")
	}

	outputPath := "./videos/test_clip.mp4"
	selectedFormat := "22"
	fileSizeLimit := int64(500000000)
	from := "00:00:30"
	to := "00:01:00"
	url := "https://www.youtube.com/watch?v=example"

	_, _ = DownloadAndCutVideo(outputPath, selectedFormat, fileSizeLimit, from, to, url)

	// Should have two command executions: yt-dlp download then ffmpeg clip
	if len(allCapturedArgs) < 2 {
		t.Errorf("Expected at least 2 command executions (yt-dlp + ffmpeg), got %d", len(allCapturedArgs))
		return
	}

	// First command should be yt-dlp for downloading
	ytdlpArgs := allCapturedArgs[0]
	if ytdlpArgs[0] != "yt-dlp" {
		t.Errorf("Expected first command to be 'yt-dlp', got '%s'", ytdlpArgs[0])
	}

	// Check for user agent presence in yt-dlp command
	userAgentFound := false
	for i, arg := range ytdlpArgs {
		if arg == "--user-agent" && i+1 < len(ytdlpArgs) {
			userAgentFound = true
			break
		}
	}
	if !userAgentFound {
		t.Error("Expected --user-agent argument to be present in yt-dlp command")
	}

	// Second command should be ffmpeg for clipping
	ffmpegArgs := allCapturedArgs[1]
	if ffmpegArgs[0] != "ffmpeg" {
		t.Errorf("Expected second command to be 'ffmpeg', got '%s'", ffmpegArgs[0])
	}

	// Check that ffmpeg has the timing arguments
	ssFound := false
	toFound := false
	for i, arg := range ffmpegArgs {
		if arg == "-ss" && i+1 < len(ffmpegArgs) && ffmpegArgs[i+1] == from {
			ssFound = true
		}
		if arg == "-to" && i+1 < len(ffmpegArgs) && ffmpegArgs[i+1] == to {
			toFound = true
		}
	}
	if !ssFound {
		t.Error("Expected -ss argument with correct timing in ffmpeg command")
	}
	if !toFound {
		t.Error("Expected -to argument with correct timing in ffmpeg command")
	}
}

func TestGetVideoDuration(t *testing.T) {
	originalExecContext := execContext
	defer func() { execContext = originalExecContext }()

	var capturedArgs []string
	execContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
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

	// Verify that yt-dlp is called with enhanced anti-detection arguments
	if len(capturedArgs) < 3 {
		t.Error("Expected more arguments but got too few")
		return
	}

	if capturedArgs[0] != "yt-dlp" {
		t.Errorf("Expected first arg to be 'yt-dlp', got '%s'", capturedArgs[0])
	}

	// Check for user agent presence
	userAgentFound := false
	for i, arg := range capturedArgs {
		if arg == "--user-agent" && i+1 < len(capturedArgs) {
			userAgentFound = true
			break
		}
	}
	if !userAgentFound {
		t.Error("Expected --user-agent argument to be present")
	}

	// Check that the original arguments are preserved
	getDurationFound := false
	for _, arg := range capturedArgs {
		if arg == "--get-duration" {
			getDurationFound = true
			break
		}
	}
	if !getDurationFound {
		t.Error("Expected --get-duration argument to be present")
	}
}
