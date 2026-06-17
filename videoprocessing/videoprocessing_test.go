package videoprocessing

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"ytclipper-go/config"
)

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func flagValue(args []string, flag string) (string, bool) {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

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
		{name: "Valid bitrate in KiB", additional: "128KiB", expected: "128KiB"},
		{name: "Valid bitrate in MiB", additional: "1.2MiB", expected: "1.2MiB"},
		{name: "Valid bitrate with tilde (~)", additional: "~192KiB", expected: "~192KiB"},
		{name: "No bitrate", additional: "Some other text", expected: "N/A"},
		{name: "Empty string", additional: "", expected: "N/A"},
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

	if len(capturedArgs) == 0 || capturedArgs[0] != "yt-dlp" {
		t.Fatalf("Expected first arg to be 'yt-dlp', got %v", capturedArgs)
	}

	if v, ok := flagValue(capturedArgs, "-o"); !ok || v != outputPath {
		t.Errorf("Expected -o %q, got %q (present=%v)", outputPath, v, ok)
	}
	if v, ok := flagValue(capturedArgs, "-f"); !ok || v != selectedFormat {
		t.Errorf("Expected -f %q, got %q (present=%v)", selectedFormat, v, ok)
	}

	expectedSection := fmt.Sprintf("*%s-%s", from, to)
	if v, ok := flagValue(capturedArgs, "--download-sections"); !ok || v != expectedSection {
		t.Errorf("Expected --download-sections %q, got %q (present=%v)", expectedSection, v, ok)
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

	// Output parsing is covered by TestExtractDuration; here we only assert the
	// command is built correctly (avoids depending on `echo` being an executable,
	// which it isn't on Windows).
	_, _ = GetVideoDuration("https://www.youtube.com/watch?v=example")

	if len(capturedArgs) == 0 || capturedArgs[0] != "yt-dlp" {
		t.Fatalf("Expected first arg to be 'yt-dlp', got %v", capturedArgs)
	}
	if !hasFlag(capturedArgs, "--get-duration") {
		t.Error("Expected --get-duration argument to be present")
	}
	if !hasFlag(capturedArgs, "--no-warnings") {
		t.Error("Expected --no-warnings argument to be present")
	}
}

// TestCommonArgsHonorsProxyAndCookies verifies that the proxy and cookies are
// applied when configured -- and that none of the old anti-detection flags leak
// back in.
func TestCommonArgsHonorsProxyAndCookies(t *testing.T) {
	original := config.CONFIG.YtDlpConfig
	defer func() { config.CONFIG.YtDlpConfig = original }()

	config.CONFIG.YtDlpConfig.Proxy = "socks5h://10.0.0.1:1080"
	config.CONFIG.YtDlpConfig.CookiesFile = "/tmp/cookies.txt"
	config.CONFIG.YtDlpConfig.CookiesContent = ""
	config.CONFIG.YtDlpConfig.ExtractorRetries = 0

	args := commonArgs()

	if v, ok := flagValue(args, "--proxy"); !ok || v != "socks5h://10.0.0.1:1080" {
		t.Errorf("Expected --proxy to be applied, got %v", args)
	}
	if v, ok := flagValue(args, "--cookies"); !ok || v != "/tmp/cookies.txt" {
		t.Errorf("Expected --cookies to be applied, got %v", args)
	}
	if hasFlag(args, "--user-agent") || hasFlag(args, "--sleep-requests") || hasFlag(args, "--add-header") {
		t.Errorf("Did not expect anti-detection flags in common args, got %v", args)
	}
}

func TestCommonArgsWithoutProxyOrCookies(t *testing.T) {
	original := config.CONFIG.YtDlpConfig
	defer func() { config.CONFIG.YtDlpConfig = original }()

	config.CONFIG.YtDlpConfig.Proxy = ""
	config.CONFIG.YtDlpConfig.CookiesFile = ""
	config.CONFIG.YtDlpConfig.CookiesContent = ""

	args := commonArgs()

	if hasFlag(args, "--proxy") {
		t.Errorf("Did not expect --proxy when unset, got %v", args)
	}
	if hasFlag(args, "--cookies") {
		t.Errorf("Did not expect --cookies when unset, got %v", args)
	}
}
