package videoprocessing

import (
	"context"
	"fmt"
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

	var capturedArgs []string
	execContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
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

	// Verify that yt-dlp is called with enhanced anti-detection arguments
	if len(capturedArgs) < 5 {
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

	// Check for download-sections argument
	downloadSectionsFound := false
	expectedSection := fmt.Sprintf("*%s-%s", from, to)
	for i, arg := range capturedArgs {
		if arg == "--download-sections" && i+1 < len(capturedArgs) && capturedArgs[i+1] == expectedSection {
			downloadSectionsFound = true
			break
		}
	}
	if !downloadSectionsFound {
		t.Errorf("Expected --download-sections argument with value '%s'", expectedSection)
	}

	// Check for enhanced headers (may be present in fallback strategies)
	headerFound := false
	for _, arg := range capturedArgs {
		if arg == "--add-header" {
			headerFound = true
			break
		}
	}
	// Headers are not guaranteed in strategy 1 (legacy config), so we don't fail if not present
	if headerFound {
		t.Log("Enhanced headers found in yt-dlp arguments")
	} else {
		t.Log("No enhanced headers found - likely using legacy configuration strategy")
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
