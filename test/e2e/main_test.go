//go:build e2e

// Package e2e drives a headless Chrome against the real app.
//
// TestMain builds the app and the fake yt-dlp and ffmpeg (test/fakes) into a
// temporary directory. Servers start on demand, on a free port, with that
// directory first on PATH, so the whole page → API → job → download flow runs
// offline and deterministically.
//
//	go test -tags e2e -count=1 ./test/e2e/
//
// Environment:
//   - E2E_BASE_URL: test an already running server instead of starting one.
//     It must have the fake yt-dlp first on PATH; tests that need their own
//     server configuration are skipped.
//   - E2E_CHROME: path of the Chrome, Chromium or Edge binary (default: found
//     by chromedp).
//   - E2E_ARTIFACTS: where failure screenshots, console logs and server logs
//     go (default test/e2e/artifacts).
//   - E2E_SCREENSHOTS: a directory; turns on TestScreenshots, which writes the
//     README images to <dir>/docs/screenshot-{light,dark}.png.
package e2e

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	repoRoot     string // the repository, where templates/ and static/ live
	binDir       string // the app and the fakes
	artifactsDir string
	coverDir     string // E2E_COVERDIR: build a -cover app and collect its coverage here

	cleanupMu sync.Mutex
	cleanups  []func()

	defaultServerOnce sync.Once
	defaultServer     *server
	defaultServerErr  error
)

func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

func runMain(m *testing.M) int {
	defer func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		log.Printf("e2e: %v", err)
		return 1
	}
	repoRoot = root

	if coverDir = os.Getenv("E2E_COVERDIR"); coverDir != "" {
		if coverDir, err = filepath.Abs(coverDir); err == nil {
			err = os.MkdirAll(coverDir, 0o755)
		}
		if err != nil {
			log.Printf("e2e: coverage directory: %v", err)
			return 1
		}
	}

	artifactsDir = os.Getenv("E2E_ARTIFACTS")
	if artifactsDir == "" {
		artifactsDir = filepath.Join(repoRoot, "test", "e2e", "artifacts")
	}
	if artifactsDir, err = filepath.Abs(artifactsDir); err == nil {
		err = os.MkdirAll(artifactsDir, 0o755)
	}
	if err != nil {
		log.Printf("e2e: artifacts directory: %v", err)
		return 1
	}

	if os.Getenv("E2E_BASE_URL") == "" || os.Getenv("E2E_SCREENSHOTS") != "" {
		dir, err := os.MkdirTemp("", "ytclipper-e2e-")
		if err != nil {
			log.Printf("e2e: %v", err)
			return 1
		}
		cleanups = append(cleanups, func() { removeAll(dir) })
		binDir = filepath.Join(dir, "bin")
		start := time.Now()
		if err := buildBinaries(binDir); err != nil {
			log.Printf("e2e: %v", err)
			return 1
		}
		log.Printf("e2e: built the app and the fakes in %v", time.Since(start).Round(time.Millisecond))
	}

	return m.Run()
}

// removeAll removes a directory, retrying for a moment: on Windows a fake
// yt-dlp that is still sleeping keeps its binary locked.
func removeAll(dir string) {
	for i := 0; i < 20; i++ {
		if err := os.RemoveAll(dir); err == nil {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func exe(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func goCommand() string {
	if p, err := exec.LookPath("go"); err == nil {
		return p
	}
	return filepath.Join(runtime.GOROOT(), "bin", exe("go"))
}

func buildBinaries(dir string) error {
	builds := []struct{ out, pkg string }{
		{exe("ytclipper"), "."},
		{exe("yt-dlp"), "./test/fakes/yt-dlp"},
		{exe("ffmpeg"), "./test/fakes/ffmpeg"},
	}
	for _, b := range builds {
		args := []string{"build", "-o", filepath.Join(dir, b.out)}
		if coverDir != "" && b.pkg == "." {
			// Coverage-instrumented app: it writes coverage data to
			// GOCOVERDIR when it exits normally (see server.stop).
			args = append(args, "-cover", "-coverpkg=./...")
		}
		cmd := exec.Command(goCommand(), append(args, b.pkg)...)
		cmd.Dir = repoRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("go build %s: %w\n%s", b.pkg, err, out)
		}
	}
	return nil
}

// server is a running app.
type server struct {
	URL     string
	LogPath string
	cmd     *exec.Cmd
	done    chan struct{}
}

// baseEnv is the configuration every test server gets: no cleanup, short
// timeouts, and a rate limit high enough that parallel tests never hit it.
func baseEnv(clipDir string) map[string]string {
	return map[string]string{
		"YTCLIPPER_CLIP_CLEANUP_SCHEDULER_ENABLED":             "false",
		"YTCLIPPER_CLIP_CLEANUP_SCHEDULER_CLIP_DIRECTORY_PATH": clipDir,
		"YTCLIPPER_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS":          "15",
		"YTCLIPPER_YT_DLP_DOWNLOAD_TIMEOUT_IN_SECONDS":         "30",
		"YTCLIPPER_YT_DLP_PROXY":                               "",
		"YTCLIPPER_RATE_LIMITER_RATE":                          "1000",
		"YTCLIPPER_RATE_LIMITER_BURST":                         "1000",
		"YTCLIPPER_MAX_CONCURRENT_JOBS":                        "4",
		"YTCLIPPER_MAX_QUEUED_JOBS":                            "",
		"YTCLIPPER_AUTH_USERNAME":                              "",
		"YTCLIPPER_AUTH_PASSWORD":                              "",
		"YTCLIPPER_TRUSTED_PROXIES":                            "",
		"YTCLIPPER_DEBUG":                                      "false",
	}
}

// startServer starts the app on a free port with the fakes first on PATH.
// extra overrides baseEnv (and sets FAKE_YTDLP_* for the fake). The server
// log goes to <artifacts>/server-<name>.log.
func startServer(name string, extra map[string]string) (*server, error) {
	if binDir == "" {
		return nil, fmt.Errorf("no binaries built (E2E_BASE_URL is set)")
	}
	clipDir, err := os.MkdirTemp("", "ytclipper-e2e-clips-")
	if err != nil {
		return nil, err
	}
	port, err := freePort()
	if err != nil {
		return nil, err
	}

	env := baseEnv(clipDir)
	env["YTCLIPPER_PORT"] = port
	for k, v := range extra {
		env[k] = v
	}
	cmdEnv := append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if coverDir != "" {
		cmdEnv = append(cmdEnv, "GOCOVERDIR="+coverDir)
	}
	for k, v := range env {
		cmdEnv = append(cmdEnv, k+"="+v)
	}

	logPath := filepath.Join(artifactsDir, "server-"+name+".log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(filepath.Join(binDir, exe("ytclipper")))
	cmd.Dir = repoRoot
	cmd.Env = cmdEnv
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	setProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("starting the server: %w", err)
	}
	s := &server{URL: "http://127.0.0.1:" + port, LogPath: logPath, cmd: cmd, done: make(chan struct{})}
	go func() {
		_ = cmd.Wait()
		logFile.Close()
		close(s.done)
	}()
	addCleanup(func() {
		s.stop()
		removeAll(clipDir)
	})

	if err := s.waitHealthy(20 * time.Second); err != nil {
		s.stop()
		logText, _ := os.ReadFile(logPath)
		return nil, fmt.Errorf("server %s: %w\n%s", name, err, logText)
	}
	return s, nil
}

func (s *server) waitHealthy(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-s.done:
			return fmt.Errorf("exited during startup")
		default:
		}
		resp, err := http.Get(s.URL + "/health")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("not healthy after %v", timeout)
}

func (s *server) stop() {
	select {
	case <-s.done:
		return
	default:
	}
	if coverDir != "" {
		// A killed process writes no coverage data: ask it to shut down
		// first, so main returns and the counters are flushed.
		terminate(s.cmd)
		select {
		case <-s.done:
		case <-time.After(10 * time.Second):
		}
	}
	killProcessGroup(s.cmd)
	select {
	case <-s.done:
	case <-time.After(10 * time.Second):
	}
}

func freePort() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer l.Close()
	_, port, err := net.SplitHostPort(l.Addr().String())
	return port, err
}

// baseURL returns the shared server: E2E_BASE_URL, or one started on first
// use.
func baseURL(t *testing.T) string {
	t.Helper()
	if v := os.Getenv("E2E_BASE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	defaultServerOnce.Do(func() {
		defaultServer, defaultServerErr = startServer("default", nil)
	})
	if defaultServerErr != nil {
		t.Fatalf("starting the server: %v", defaultServerErr)
	}
	return defaultServer.URL
}

// ownServer starts a server with its own configuration for one test. It
// skips the test against E2E_BASE_URL, whose configuration is unknown.
func ownServer(t *testing.T, name string, extra map[string]string) *server {
	t.Helper()
	if os.Getenv("E2E_BASE_URL") != "" && binDir == "" {
		t.Skip("needs its own server; E2E_BASE_URL is set")
	}
	s, err := startServer(name, extra)
	if err != nil {
		t.Fatalf("starting the %s server: %v", name, err)
	}
	t.Cleanup(s.stop)
	return s
}

// httpGet fetches a URL from the test process (not the browser).
func httpGet(url string) (*http.Response, []byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	return resp, buf.Bytes(), err
}

// addCleanup registers f to run when TestMain ends.
func addCleanup(f func()) {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()
	cleanups = append(cleanups, f)
}
