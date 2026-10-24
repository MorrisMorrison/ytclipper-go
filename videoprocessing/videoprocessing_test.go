package videoprocessing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"ytclipper-go/config"
	"ytclipper-go/jobs"
)

func hasFlag(args []string, flag string) bool {
	return slices.Contains(args, flag)
}

func flagValue(args []string, flag string) (string, bool) {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

// fakeYtDlp replaces execContext with a command that re-runs this test binary
// as TestHelperProcess, which acts like yt-dlp: in "info" mode it prints
// output, in "download" mode it writes a file to the -o template. Every call's
// arguments are appended to calls. The original is restored on cleanup.
func fakeYtDlp(t *testing.T, mode string, output string) *[][]string {
	t.Helper()
	original := execContext
	t.Cleanup(func() { execContext = original })

	calls := &[][]string{}
	execContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		*calls = append(*calls, append([]string{name}, arg...))
		cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestHelperProcess$", "--"}, arg...)...)
		cmd.Env = append(os.Environ(),
			"YTCLIPPER_TEST_HELPER=1",
			"YTCLIPPER_TEST_HELPER_MODE="+mode,
			"YTCLIPPER_TEST_HELPER_OUTPUT="+output,
		)
		return cmd
	}
	return calls
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("YTCLIPPER_TEST_HELPER") != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}

	switch os.Getenv("YTCLIPPER_TEST_HELPER_MODE") {
	case "info":
		fmt.Print(os.Getenv("YTCLIPPER_TEST_HELPER_OUTPUT"))
		os.Exit(0)
	case "download":
		if out, ok := flagValue(args, "-o"); ok {
			if err := os.WriteFile(strings.ReplaceAll(out, "%(ext)s", "mp4"), []byte("clip"), 0o644); err != nil {
				os.Exit(2)
			}
		}
		os.Exit(0)
	default:
		fmt.Println(os.Getenv("YTCLIPPER_TEST_HELPER_OUTPUT"))
		os.Exit(1)
	}
}

// useFreshInfoCache gives the test an empty video info cache.
func useFreshInfoCache(t *testing.T) {
	t.Helper()
	original := videoInfoCache
	videoInfoCache = newInfoCache(infoCacheTTL, infoCacheMaxSize)
	t.Cleanup(func() { videoInfoCache = original })
}

func TestDownloadAndCutVideo(t *testing.T) {
	calls := fakeYtDlp(t, "download", "")

	outputPath := filepath.Join(t.TempDir(), "test_clip.%(ext)s")
	from := "00:00:30"
	to := "00:01:00"
	url := "https://www.youtube.com/watch?v=example"

	if _, err := DownloadAndCutVideo(outputPath, []string{"-f", "22"}, 500000000, from, to, url); err != nil {
		t.Fatalf("DownloadAndCutVideo: %v", err)
	}

	args := (*calls)[0]
	if args[0] != "yt-dlp" {
		t.Fatalf("Expected first arg to be 'yt-dlp', got %v", args)
	}
	if v, ok := flagValue(args, "-o"); !ok || v != outputPath {
		t.Errorf("Expected -o %q, got %q (present=%v)", outputPath, v, ok)
	}
	if v, ok := flagValue(args, "-f"); !ok || v != "22" {
		t.Errorf("Expected -f 22, got %q (present=%v)", v, ok)
	}
	if v, ok := flagValue(args, "--download-sections"); !ok || v != "*00:00:30-00:01:00" {
		t.Errorf("Expected --download-sections *00:00:30-00:01:00, got %q (present=%v)", v, ok)
	}
	if args[len(args)-1] != url {
		t.Errorf("Expected the URL last, got %v", args)
	}
}

func TestFormatArgsPresets(t *testing.T) {
	useFreshInfoCache(t)
	calls := fakeYtDlp(t, "fail", "ERROR: should not be called")

	video := formatArgs(ClipRequest{VideoID: "dQw4w9WgXcQ", Preset: PresetVideo})
	wantVideo := []string{"-f", "bv*[vcodec^=avc1][height<=1080]+ba[acodec^=mp4a]/b[vcodec^=avc1][height<=1080]/bv*[height<=1080]+ba/b", "--merge-output-format", "mp4"}
	if !slices.Equal(video, wantVideo) {
		t.Errorf("video preset args = %v; want %v", video, wantVideo)
	}

	audio := formatArgs(ClipRequest{VideoID: "dQw4w9WgXcQ", Preset: PresetAudio})
	wantAudio := []string{"-f", "ba", "-x", "--audio-format", "mp3", "--audio-quality", "192K"}
	if !slices.Equal(audio, wantAudio) {
		t.Errorf("audio preset args = %v; want %v", audio, wantAudio)
	}

	merged := formatArgs(ClipRequest{VideoID: "dQw4w9WgXcQ", Format: "137+140"})
	if !slices.Equal(merged, []string{"-f", "137+140"}) {
		t.Errorf("merged format args = %v", merged)
	}

	if len(*calls) != 0 {
		t.Errorf("presets and merged formats must not call yt-dlp, got %v", *calls)
	}
}

func TestFormatArgsAddsAudioToVideoOnlyFormat(t *testing.T) {
	useFreshInfoCache(t)
	calls := fakeYtDlp(t, "info", infoFixture)

	tests := map[string]string{
		"137": "137+bestaudio", // video only
		"136": "136+bestaudio", // video only
		"18":  "18",            // audio and video
		"140": "140",           // audio only
		"999": "999",           // unknown: yt-dlp reports it
	}
	for format, want := range tests {
		got := formatArgs(ClipRequest{VideoID: "dQw4w9WgXcQ", Format: format})
		if !slices.Equal(got, []string{"-f", want}) {
			t.Errorf("formatArgs(%s) = %v; want -f %s", format, got, want)
		}
	}
	if len(*calls) != 1 {
		t.Errorf("expected one cached yt-dlp -J call, got %d", len(*calls))
	}
}

func TestFormatArgsKeepsFormatWhenInfoFails(t *testing.T) {
	useFreshInfoCache(t)
	fakeYtDlp(t, "fail", "ERROR: [youtube] dQw4w9WgXcQ: Video unavailable")

	got := formatArgs(ClipRequest{VideoID: "dQw4w9WgXcQ", Format: "137"})
	if !slices.Equal(got, []string{"-f", "137"}) {
		t.Errorf("formatArgs = %v; want -f 137", got)
	}
}

func TestProcessClip(t *testing.T) {
	useFreshInfoCache(t)
	original := config.CONFIG.ClipCleanUpSchedulerConfig.ClipDirectoryPath
	config.CONFIG.ClipCleanUpSchedulerConfig.ClipDirectoryPath = t.TempDir()
	t.Cleanup(func() { config.CONFIG.ClipCleanUpSchedulerConfig.ClipDirectoryPath = original })

	fakeYtDlp(t, "info", infoFixture)
	if _, err := GetVideoInfo("dQw4w9WgXcQ"); err != nil {
		t.Fatal(err)
	}

	calls := fakeYtDlp(t, "download", "")
	job := jobs.NewJob()
	ProcessClip(job.ID, ClipRequest{VideoID: "dQw4w9WgXcQ", From: "00:00:10", To: "01:02:03", Format: "137"})

	got, _ := jobs.GetJobById(job.ID)
	if got.Status != jobs.StatusCompleted {
		t.Fatalf("job status = %s (%s); want completed", got.Status, got.Error)
	}
	if want := "Never Gonna Give You Up [0-00-10 - 1-02-03].mp4"; got.FileName != want {
		t.Errorf("FileName = %q; want %q", got.FileName, want)
	}
	if filepath.Base(got.FilePath) != job.ID+".mp4" {
		t.Errorf("FilePath = %q", got.FilePath)
	}
	args := (*calls)[0]
	if v, _ := flagValue(args, "-f"); v != "137+bestaudio" {
		t.Errorf("-f = %q; want 137+bestaudio", v)
	}
	if args[len(args)-1] != "https://www.youtube.com/watch?v=dQw4w9WgXcQ" {
		t.Errorf("URL = %q", args[len(args)-1])
	}
}

func TestProcessClipFailure(t *testing.T) {
	useFreshInfoCache(t)
	original := config.CONFIG.ClipCleanUpSchedulerConfig.ClipDirectoryPath
	config.CONFIG.ClipCleanUpSchedulerConfig.ClipDirectoryPath = t.TempDir()
	t.Cleanup(func() { config.CONFIG.ClipCleanUpSchedulerConfig.ClipDirectoryPath = original })

	fakeYtDlp(t, "fail", "[youtube] x\nERROR: [youtube] dQw4w9WgXcQ: Sign in to confirm your age")
	job := jobs.NewJob()
	ProcessClip(job.ID, ClipRequest{VideoID: "dQw4w9WgXcQ", From: "00:00:10", To: "00:00:20", Preset: PresetAudio})

	got, _ := jobs.GetJobById(job.ID)
	if got.Status != jobs.StatusError || got.Error != "[youtube] dQw4w9WgXcQ: Sign in to confirm your age" {
		t.Errorf("job = %s %q", got.Status, got.Error)
	}
}

// TestCommonArgsHonorsProxy verifies that the proxy is applied when configured --
// and that neither cookies (support removed) nor the old anti-detection flags
// leak back in.
func TestCommonArgsHonorsProxy(t *testing.T) {
	original := config.CONFIG.YtDlpConfig
	defer func() { config.CONFIG.YtDlpConfig = original }()

	config.CONFIG.YtDlpConfig.Proxy = "socks5h://10.0.0.1:1080"
	config.CONFIG.YtDlpConfig.ExtractorRetries = 0

	args := commonArgs()

	if v, ok := flagValue(args, "--proxy"); !ok || v != "socks5h://10.0.0.1:1080" {
		t.Errorf("Expected --proxy to be applied, got %v", args)
	}
	if hasFlag(args, "--cookies") {
		t.Errorf("Did not expect --cookies (cookie support removed), got %v", args)
	}
	if hasFlag(args, "--user-agent") || hasFlag(args, "--sleep-requests") || hasFlag(args, "--add-header") {
		t.Errorf("Did not expect anti-detection flags in common args, got %v", args)
	}
}

func TestCommonArgsWithoutProxy(t *testing.T) {
	original := config.CONFIG.YtDlpConfig
	defer func() { config.CONFIG.YtDlpConfig = original }()

	config.CONFIG.YtDlpConfig.Proxy = ""

	args := commonArgs()

	if hasFlag(args, "--proxy") {
		t.Errorf("Did not expect --proxy when unset, got %v", args)
	}
}

func TestSummarizeError(t *testing.T) {
	output := []byte("[youtube] abc: Downloading webpage\nERROR: [youtube] abc: Video unavailable\n")
	if got := summarizeError(output, errors.New("exit 1")); got != "[youtube] abc: Video unavailable" {
		t.Errorf("unexpected summary %q", got)
	}
	if got := summarizeError(nil, fmt.Errorf("%w after 1s", errTimeout)); got != "Timed out while downloading the clip" {
		t.Errorf("unexpected timeout summary %q", got)
	}
	if got := summarizeError([]byte("--proxy socks5h://secret:1080"), errors.New("exit 1")); strings.Contains(got, "secret") {
		t.Errorf("summary leaked command output: %q", got)
	}
}

func TestFindOutputFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := findOutputFile(dir, "job1"); err == nil {
		t.Error("expected an error when nothing was written")
	}

	for _, name := range []string{"job1.mp4.part", "job1.mp4", "job2.webm"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := findOutputFile(dir, "job1")
	if err != nil || filepath.Base(got) != "job1.mp4" {
		t.Errorf("findOutputFile = %q, %v; want job1.mp4", got, err)
	}
}
