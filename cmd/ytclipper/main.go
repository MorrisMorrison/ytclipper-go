// Command ytclipper is the ytclipper server: it checks for yt-dlp and ffmpeg,
// starts the clip cleanup scheduler and serves the page and the API until it
// gets SIGINT or SIGTERM.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MorrisMorrison/ytclipper-go/internal/config"
	"github.com/MorrisMorrison/ytclipper-go/internal/scheduler"
	"github.com/MorrisMorrison/ytclipper-go/internal/server"
)

func checkCommand(command string) error {
	_, err := exec.LookPath(command)
	if err != nil {
		return fmt.Errorf("%s is not installed or not in PATH", command)
	}
	return nil
}

func checkDependencies() {
	slog.Info("Checking dependencies")

	for _, command := range []string{"ffmpeg", "yt-dlp"} {
		if err := checkCommand(command); err != nil {
			slog.Error("Dependency check failed", "err", err)
			os.Exit(1)
		}
	}

	slog.Info("All dependencies are installed")
}

// redactProxy hides credentials in a proxy URL before it is logged.
func redactProxy(proxy string) string {
	u, err := url.Parse(proxy)
	if err != nil || u.User == nil {
		return proxy
	}
	u.User = url.User("***")
	return u.String()
}

// isSocksProxy reports whether yt-dlp's proxy is a SOCKS proxy. yt-dlp can use
// it, but ffmpeg (which downloads --download-sections) cannot.
func isSocksProxy(proxy string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(proxy)), "socks")
}

// newLogger returns the process logger: text or JSON on stdout, at debug level
// when debug is set.
func newLogger(format string, debug bool) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if debug {
		opts.Level = slog.LevelDebug
	}
	if strings.EqualFold(format, "json") {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

func main() {
	slog.SetDefault(newLogger(config.CONFIG.LogFormat, config.CONFIG.Debug))
	slog.Info("Start ytclipper")
	checkDependencies()

	if proxy := config.CONFIG.YtDlpConfig.Proxy; proxy != "" {
		slog.Info("yt-dlp proxy", "proxy", redactProxy(proxy))
		if isSocksProxy(proxy) {
			slog.Warn("yt-dlp proxy is SOCKS: clips are downloaded by ffmpeg, which can't use SOCKS, " +
				"so YouTube will usually answer 403. Use an HTTP proxy (http://host:port) instead.")
		}
	}

	scheduler.StartClipCleanUpScheduler()
	e, err := server.New()
	if err != nil {
		slog.Error("Invalid configuration", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("Starting server", "port", config.CONFIG.Port)
		if err := e.Start(":" + config.CONFIG.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Fatal(err)
		}
	}()

	<-ctx.Done()
	slog.Info("Shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatal(err)
	}
}
