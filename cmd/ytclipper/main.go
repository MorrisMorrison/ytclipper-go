// Command ytclipper is the ytclipper server: it checks for yt-dlp and ffmpeg,
// starts the clip cleanup scheduler and serves the page and the API until it
// gets SIGINT or SIGTERM.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
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

	"github.com/MorrisMorrison/gutils/glogger"
)

func checkCommand(command string) error {
	_, err := exec.LookPath(command)
	if err != nil {
		return fmt.Errorf("%s is not installed or not in PATH", command)
	}
	return nil
}

func checkDependencies() {
	glogger.Log.Info("Checking dependencies")

	for _, command := range []string{"ffmpeg", "yt-dlp"} {
		if err := checkCommand(command); err != nil {
			log.Fatalf("Dependency check failed: %v", err)
		}
	}

	glogger.Log.Info("All dependencies are installed.")
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

func main() {
	glogger.Log.Info("Start ytclipper")
	checkDependencies()

	if proxy := config.CONFIG.YtDlpConfig.Proxy; proxy != "" {
		glogger.Log.Infof("yt-dlp proxy: %s", redactProxy(proxy))
		if isSocksProxy(proxy) {
			glogger.Log.Warning("yt-dlp proxy is SOCKS: clips are downloaded by ffmpeg, which can't use SOCKS, " +
				"so YouTube will usually answer 403. Use an HTTP proxy (http://host:port) instead.")
		}
	}

	scheduler.StartClipCleanUpScheduler()
	e, err := server.New()
	if err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		glogger.Log.Infof("Starting server on port %s", config.CONFIG.Port)
		if err := e.Start(":" + config.CONFIG.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Fatal(err)
		}
	}()

	<-ctx.Done()
	glogger.Log.Info("Shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatal(err)
	}
}
