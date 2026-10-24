//go:build unix

package videoprocessing

import (
	"errors"
	"testing"
	"time"
)

// A child that inherits the output pipes (like the ffmpeg yt-dlp starts for
// --download-sections) must not keep a timed-out command running: the whole
// process group is killed. waitDelay is raised so that only the group kill can
// make this return quickly.
func TestTimeoutKillsChildProcesses(t *testing.T) {
	original := waitDelay
	waitDelay = time.Minute
	t.Cleanup(func() { waitDelay = original })

	fakeYtDlp(t, "spawn", "")
	start := time.Now()
	_, err := executeWithTimeout(500*time.Millisecond, "yt-dlp", "--download-sections", "*00:00:01-00:00:02")
	if !errors.Is(err, errTimeout) {
		t.Fatalf("err = %v; want a timeout", err)
	}
	if elapsed := time.Since(start); elapsed > 15*time.Second {
		t.Errorf("returned after %v; the child kept the command alive", elapsed)
	}
}
