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
	"time"
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
// as TestHelperProcess, which acts like yt-dlp (see there for the modes).
// extraEnv is added to the helper's environment. Every call's arguments are
// appended to calls. The original is restored on cleanup.
func fakeYtDlp(t *testing.T, mode string, output string, extraEnv ...string) *[][]string {
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
		cmd.Env = append(cmd.Env, extraEnv...)
		return cmd
	}
	return calls
}

// TestHelperProcess is the fake yt-dlp. Modes:
//   - info: prints YTCLIPPER_TEST_HELPER_OUTPUT to stdout and
//     YTCLIPPER_TEST_HELPER_STDERR to stderr
//   - download: answers -J like info; otherwise writes "clip" to the -o
//     template (with mp4 as extension)
//   - sleep: sleeps for a minute
//   - spawn: starts a child in sleep mode that shares its output pipes (like
//     the ffmpeg yt-dlp starts), then sleeps for a minute
//   - anything else: prints the output to stderr and exits with 1
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

	switch mode := os.Getenv("YTCLIPPER_TEST_HELPER_MODE"); {
	case mode == "info" || (mode == "download" && hasFlag(args, "-J")):
		fmt.Fprint(os.Stderr, os.Getenv("YTCLIPPER_TEST_HELPER_STDERR"))
		fmt.Print(os.Getenv("YTCLIPPER_TEST_HELPER_OUTPUT"))
		os.Exit(0)
	case mode == "download":
		if out, ok := flagValue(args, "-o"); ok {
			if err := os.WriteFile(strings.ReplaceAll(out, "%(ext)s", "mp4"), []byte("clip"), 0o644); err != nil {
				os.Exit(2)
			}
		}
		os.Exit(0)
	case mode == "sleep":
		time.Sleep(time.Minute)
		os.Exit(0)
	case mode == "spawn":
		child := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
		child.Env = append(os.Environ(), "YTCLIPPER_TEST_HELPER_MODE=sleep")
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err := child.Start(); err != nil {
			os.Exit(3)
		}
		time.Sleep(time.Minute)
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, os.Getenv("YTCLIPPER_TEST_HELPER_OUTPUT"))
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

// useClipDir points the clip directory at a fresh temp dir and returns it.
func useClipDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	original := config.CONFIG.ClipCleanUpSchedulerConfig.ClipDirectoryPath
	config.CONFIG.ClipCleanUpSchedulerConfig.ClipDirectoryPath = dir
	t.Cleanup(func() { config.CONFIG.ClipCleanUpSchedulerConfig.ClipDirectoryPath = original })
	return dir
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
	// yt-dlp downloads sections with ffmpeg, which ignores --max-filesize.
	if v, ok := flagValue(args, "--downloader-args"); !ok || v != "ffmpeg:-fs 500000000" {
		t.Errorf("Expected --downloader-args ffmpeg:-fs 500000000, got %q (present=%v)", v, ok)
	}
	if v, ok := flagValue(args, "--max-filesize"); !ok || v != "500000000" {
		t.Errorf("Expected --max-filesize 500000000, got %q (present=%v)", v, ok)
	}
	if args[len(args)-1] != url {
		t.Errorf("Expected the URL last, got %v", args)
	}
}

func TestFormatArgsPresets(t *testing.T) {
	video := formatArgs(ClipRequest{VideoID: "dQw4w9WgXcQ", Preset: PresetVideo}, nil)
	wantVideo := []string{"-f", "bv*[vcodec^=avc1][height<=1080]+ba[acodec^=mp4a]/b[vcodec^=avc1][height<=1080]/bv*[height<=1080]+ba/b", "--merge-output-format", "mp4"}
	if !slices.Equal(video, wantVideo) {
		t.Errorf("video preset args = %v; want %v", video, wantVideo)
	}

	audio := formatArgs(ClipRequest{VideoID: "dQw4w9WgXcQ", Preset: PresetAudio}, nil)
	wantAudio := []string{"-f", "ba/b", "-x", "--audio-format", "mp3", "--audio-quality", "192K"}
	if !slices.Equal(audio, wantAudio) {
		t.Errorf("audio preset args = %v; want %v", audio, wantAudio)
	}

	merged := formatArgs(ClipRequest{VideoID: "dQw4w9WgXcQ", Format: "137+140"}, nil)
	if !slices.Equal(merged, []string{"-f", "137+140"}) {
		t.Errorf("merged format args = %v", merged)
	}
}

func TestFormatArgsAddsAudioToVideoOnlyFormat(t *testing.T) {
	info, err := parseVideoInfo([]byte(infoFixture))
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]string{
		"137": "137+bestaudio[ext=m4a]/137+bestaudio", // mp4 video only: prefer m4a audio
		"136": "136+bestaudio[ext=m4a]/136+bestaudio", // mp4 video only
		"247": "247+bestaudio",                        // webm video only
		"18":  "18",                                   // audio and video
		"140": "140",                                  // audio only
		"999": "999",                                  // unknown: yt-dlp reports it
	}
	for format, want := range tests {
		got := formatArgs(ClipRequest{VideoID: "dQw4w9WgXcQ", Format: format}, info)
		if !slices.Equal(got, []string{"-f", want}) {
			t.Errorf("formatArgs(%s) = %v; want -f %s", format, got, want)
		}
	}
}

func TestFormatArgsKeepsFormatWithoutInfo(t *testing.T) {
	got := formatArgs(ClipRequest{VideoID: "dQw4w9WgXcQ", Format: "137"}, nil)
	if !slices.Equal(got, []string{"-f", "137"}) {
		t.Errorf("formatArgs = %v; want -f 137", got)
	}
}

func TestProcessClip(t *testing.T) {
	useFreshInfoCache(t)
	useClipDir(t)

	fakeYtDlp(t, "info", infoFixture)
	if _, err := GetVideoInfo("dQw4w9WgXcQ"); err != nil {
		t.Fatal(err)
	}

	calls := fakeYtDlp(t, "download", "")
	job := jobs.NewJob()
	ProcessClip(job.ID, ClipRequest{VideoID: "dQw4w9WgXcQ", From: "00:00:10", To: "00:02:03", Format: "137"})

	got, _ := jobs.GetJobById(job.ID)
	if got.Status != jobs.StatusCompleted {
		t.Fatalf("job status = %s (%s); want completed", got.Status, got.Error)
	}
	if want := "Never Gonna Give You Up [0-00-10 - 0-02-03].mp4"; got.FileName != want {
		t.Errorf("FileName = %q; want %q", got.FileName, want)
	}
	if filepath.Base(got.FilePath) != job.ID+".mp4" {
		t.Errorf("FilePath = %q", got.FilePath)
	}
	if len(*calls) != 1 {
		t.Fatalf("expected only the download (info is cached), got %v", *calls)
	}
	args := (*calls)[0]
	if v, _ := flagValue(args, "-f"); v != "137+bestaudio[ext=m4a]/137+bestaudio" {
		t.Errorf("-f = %q; want 137+bestaudio[ext=m4a]/137+bestaudio", v)
	}
	if args[len(args)-1] != "https://www.youtube.com/watch?v=dQw4w9WgXcQ" {
		t.Errorf("URL = %q", args[len(args)-1])
	}
}

// The title must not depend on the page having warmed the cache.
func TestProcessClipLoadsInfoItself(t *testing.T) {
	useFreshInfoCache(t)
	useClipDir(t)

	calls := fakeYtDlp(t, "download", infoFixture)
	job := jobs.NewJob()
	ProcessClip(job.ID, ClipRequest{VideoID: "dQw4w9WgXcQ", From: "00:00:10", To: "00:00:20", Preset: PresetVideo})

	got, _ := jobs.GetJobById(job.ID)
	if got.Status != jobs.StatusCompleted || got.FileName != "Never Gonna Give You Up [0-00-10 - 0-00-20].mp4" {
		t.Errorf("job = %s %q %q", got.Status, got.FileName, got.Error)
	}
	if len(*calls) != 2 || !hasFlag((*calls)[0], "-J") {
		t.Errorf("expected -J then the download, got %v", *calls)
	}
}

func TestProcessClipRejectsEndAfterVideo(t *testing.T) {
	useFreshInfoCache(t)
	useClipDir(t)

	calls := fakeYtDlp(t, "download", infoFixture) // duration 212 s
	job := jobs.NewJob()
	ProcessClip(job.ID, ClipRequest{VideoID: "dQw4w9WgXcQ", From: "00:03:00", To: "00:03:33", Preset: PresetVideo})

	got, _ := jobs.GetJobById(job.ID)
	if got.Status != jobs.StatusError || got.Error != "The end time is after the end of the video (3:32)." {
		t.Errorf("job = %s %q", got.Status, got.Error)
	}
	if len(*calls) != 1 {
		t.Errorf("must not download past the end, got %v", *calls)
	}
}

func TestProcessClipFailsAtSizeLimit(t *testing.T) {
	useFreshInfoCache(t)
	dir := useClipDir(t)
	original := config.CONFIG.YtDlpConfig.ClipSizeInMb
	config.CONFIG.YtDlpConfig.ClipSizeInMb = 4 // the fake writes 4 bytes, as ffmpeg -fs stops at the limit
	t.Cleanup(func() { config.CONFIG.YtDlpConfig.ClipSizeInMb = original })

	fakeYtDlp(t, "download", infoFixture)
	job := jobs.NewJob()
	ProcessClip(job.ID, ClipRequest{VideoID: "dQw4w9WgXcQ", From: "00:00:10", To: "00:00:20", Preset: PresetVideo})

	got, _ := jobs.GetJobById(job.ID)
	if got.Status != jobs.StatusError || got.Error != sizeLimitMessage(4) {
		t.Errorf("job = %s %q", got.Status, got.Error)
	}
	if _, err := findOutputFile(dir, job.ID); err == nil {
		t.Error("the truncated clip must be deleted")
	}
	if sizeLimitMessage(300*1024*1024) != "The clip is larger than the 300 MB limit. Choose a shorter range or a smaller format." {
		t.Errorf("unexpected message %q", sizeLimitMessage(300*1024*1024))
	}
}

func TestProcessClipFailure(t *testing.T) {
	useFreshInfoCache(t)
	useClipDir(t)

	fakeYtDlp(t, "fail", "[youtube] x\nERROR: [youtube] dQw4w9WgXcQ: Sign in to confirm your age")
	job := jobs.NewJob()
	ProcessClip(job.ID, ClipRequest{VideoID: "dQw4w9WgXcQ", From: "00:00:10", To: "00:00:20", Preset: PresetAudio})

	got, _ := jobs.GetJobById(job.ID)
	if got.Status != jobs.StatusError || got.Error != "[youtube] dQw4w9WgXcQ: Sign in to confirm your age" {
		t.Errorf("job = %s %q", got.Status, got.Error)
	}
}

func TestProcessClipSkipsJobsNoLongerQueued(t *testing.T) {
	useFreshInfoCache(t)
	useClipDir(t)

	calls := fakeYtDlp(t, "download", infoFixture)
	job := jobs.NewJob()
	jobs.FailJob(job.ID, "too busy")
	ProcessClip(job.ID, ClipRequest{VideoID: "dQw4w9WgXcQ", From: "00:00:10", To: "00:00:20", Preset: PresetVideo})

	if len(*calls) != 0 {
		t.Errorf("a failed job must not run, got %v", *calls)
	}
	if got, _ := jobs.GetJobById(job.ID); got.Status != jobs.StatusError || got.Error != "too busy" {
		t.Errorf("job = %s %q", got.Status, got.Error)
	}
}

func TestDownloadUsesDownloadTimeout(t *testing.T) {
	original := config.CONFIG.YtDlpConfig
	t.Cleanup(func() { config.CONFIG.YtDlpConfig = original })
	config.CONFIG.YtDlpConfig.CommandTimeoutInSeconds = 60
	config.CONFIG.YtDlpConfig.DownloadTimeoutInSeconds = 1

	fakeYtDlp(t, "sleep", "")
	start := time.Now()
	_, err := DownloadAndCutVideo(filepath.Join(t.TempDir(), "x.%(ext)s"), nil, 1, "00:00:01", "00:00:02", WatchURL("dQw4w9WgXcQ"))
	if !errors.Is(err, errTimeout) {
		t.Fatalf("err = %v; want a timeout", err)
	}
	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Errorf("took %v; the 1 s download timeout was not applied", elapsed)
	}
}

func TestExecuteSeparatesStdoutAndStderr(t *testing.T) {
	fakeYtDlp(t, "info", `{"id": "x"}`, "YTCLIPPER_TEST_HELPER_STDERR=WARNING: noise\n")
	out, err := executeWithTimeout(30*time.Second, "yt-dlp", "-J")
	if err != nil {
		t.Fatal(err)
	}
	if string(out.Stdout) != `{"id": "x"}` || string(out.Stderr) != "WARNING: noise\n" {
		t.Errorf("stdout = %q, stderr = %q", out.Stdout, out.Stderr)
	}
	if got := string(out.combined()); got != "{\"id\": \"x\"}\nWARNING: noise\n" {
		t.Errorf("combined = %q", got)
	}
}

func TestGetVideoInfoParsesStdoutOnly(t *testing.T) {
	useFreshInfoCache(t)
	// A JSON-looking line on stderr must not be mistaken for the info.
	fakeYtDlp(t, "info", infoFixture, "YTCLIPPER_TEST_HELPER_STDERR=[debug] x\n{\"title\": \"wrong\"}\n")

	info, err := GetVideoInfo("dQw4w9WgXcQ")
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "Never Gonna Give You Up" {
		t.Errorf("Title = %q", info.Title)
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

func TestFormatDuration(t *testing.T) {
	for seconds, want := range map[int]string{0: "0:00", 59: "0:59", 212: "3:32", 3600: "1:00:00", 3723: "1:02:03"} {
		if got := formatDuration(seconds); got != want {
			t.Errorf("formatDuration(%d) = %q; want %q", seconds, got, want)
		}
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
