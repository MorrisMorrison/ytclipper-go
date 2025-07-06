package videoprocessing

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
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

	// Verify that yt-dlp is called with basic arguments
	if len(capturedArgs) < 5 {
		t.Error("Expected more arguments but got too few")
		return
	}

	if capturedArgs[0] != "yt-dlp" {
		t.Errorf("Expected first arg to be 'yt-dlp', got '%s'", capturedArgs[0])
	}

	// Check for user agent presence (always required in simplified approach)
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

	// Check for anti-bot detection arguments
	sleepRequestsFound := false
	for i, arg := range capturedArgs {
		if arg == "--sleep-requests" && i+1 < len(capturedArgs) {
			sleepRequestsFound = true
			break
		}
	}
	if !sleepRequestsFound {
		t.Error("Expected --sleep-requests argument to be present")
	}

	// Verify that output path and format are included in the base arguments
	outputPathFound := false
	formatFound := false
	for i, arg := range capturedArgs {
		if arg == "-o" && i+1 < len(capturedArgs) && capturedArgs[i+1] == outputPath {
			outputPathFound = true
		}
		if arg == "-f" && i+1 < len(capturedArgs) && capturedArgs[i+1] == selectedFormat {
			formatFound = true
		}
	}
	if !outputPathFound {
		t.Error("Expected output path argument to be present")
	}
	if !formatFound {
		t.Error("Expected format argument to be present")
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

	// Verify that yt-dlp is called with basic anti-detection arguments
	if len(capturedArgs) < 3 {
		t.Error("Expected more arguments but got too few")
		return
	}

	if capturedArgs[0] != "yt-dlp" {
		t.Errorf("Expected first arg to be 'yt-dlp', got '%s'", capturedArgs[0])
	}

	// Check for user agent presence (always required in simplified approach)
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
	noWarningsFound := false
	for _, arg := range capturedArgs {
		if arg == "--get-duration" {
			getDurationFound = true
		}
		if arg == "--no-warnings" {
			noWarningsFound = true
		}
	}
	if !getDurationFound {
		t.Error("Expected --get-duration argument to be present")
	}
	if !noWarningsFound {
		t.Error("Expected --no-warnings argument to be present")
	}

	// Check for anti-bot detection arguments
	sleepRequestsFound := false
	for i, arg := range capturedArgs {
		if arg == "--sleep-requests" && i+1 < len(capturedArgs) {
			sleepRequestsFound = true
			break
		}
	}
	if !sleepRequestsFound {
		t.Error("Expected --sleep-requests argument to be present")
	}
}

func TestGetUserAgent(t *testing.T) {
	// Test that getUserAgent returns a valid user agent string
	userAgent := getUserAgent()
	if userAgent == "" {
		t.Error("Expected non-empty user agent")
	}

	// Should contain common browser identifiers
	if !strings.Contains(userAgent, "Mozilla") {
		t.Error("Expected user agent to contain Mozilla")
	}

	// Test multiple calls return different agents (when rotation is enabled)
	userAgent1 := getUserAgent()
	userAgent2 := getUserAgent()
	// Note: We can't guarantee they're different due to randomization,
	// but we can verify both are valid
	if userAgent1 == "" || userAgent2 == "" {
		t.Error("Expected valid user agents from multiple calls")
	}
}

func TestApplyAntiDetectionArgs(t *testing.T) {
	baseArgs := []string{"-f", "22", "https://example.com"}

	// Test with basic configuration
	enhancedArgs := applyAntiDetectionArgs(baseArgs)

	// Should contain user agent
	userAgentFound := false
	for i, arg := range enhancedArgs {
		if arg == "--user-agent" && i+1 < len(enhancedArgs) {
			userAgentFound = true
			break
		}
	}
	if !userAgentFound {
		t.Error("Expected --user-agent argument in enhanced args")
	}

	// Should contain sleep arguments
	sleepFound := false
	for _, arg := range enhancedArgs {
		if arg == "--sleep-requests" {
			sleepFound = true
			break
		}
	}
	if !sleepFound {
		t.Error("Expected --sleep-requests argument in enhanced args")
	}

	// Should preserve original arguments
	for _, baseArg := range baseArgs {
		found := false
		for _, enhancedArg := range enhancedArgs {
			if enhancedArg == baseArg {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected original argument '%s' to be preserved", baseArg)
		}
	}
}

func TestExecuteWithFallbackMock(t *testing.T) {
	originalExecContext := execContext
	defer func() { execContext = originalExecContext }()

	callCount := 0
	var capturedArgs [][]string

	// Mock exec function that fails on first two calls, succeeds on third
	execContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		capturedArgs = append(capturedArgs, append([]string{name}, arg...))
		callCount++

		if callCount <= 2 {
			// First two calls fail (legacy + enhanced strategies)
			return exec.Command("false") // Command that always fails
		} else {
			// Third call succeeds (basic fallback)
			return exec.Command("echo", "success")
		}
	}

	baseArgs := []string{"-f", "22", "https://example.com"}
	output, err := executeWithFallback("yt-dlp", baseArgs)

	if err != nil {
		t.Errorf("Expected executeWithFallback to succeed with fallback, but got error: %v", err)
	}

	if strings.TrimSpace(string(output)) != "success" {
		t.Errorf("Expected output 'success', but got: %s", string(output))
	}

	// Should have made exactly 3 calls (legacy + enhanced + basic)
	if callCount != 3 {
		t.Errorf("Expected 3 calls (legacy + enhanced + basic fallback), but got %d", callCount)
	}

	// Verify all calls were made
	if len(capturedArgs) != 3 {
		t.Errorf("Expected 3 captured argument sets, but got %d", len(capturedArgs))
	}

	// First call should have anti-detection args (legacy)
	// Second call should have enhanced fallback args
	// Third call should have basic fallback args (fewest)
	if len(capturedArgs[0]) <= len(capturedArgs[2]) {
		t.Error("Expected first call (legacy) to have more arguments than third call (basic)")
	}
	if len(capturedArgs[1]) <= len(capturedArgs[2]) {
		t.Error("Expected second call (enhanced) to have more arguments than third call (basic)")
	}
}
