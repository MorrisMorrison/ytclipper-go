package scheduler

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/MorrisMorrison/ytclipper-go/internal/config"
	"github.com/MorrisMorrison/ytclipper-go/internal/jobs"
)

// queuedJobTimeout is how long a job may wait for a worker before cleanup
// fails it.
const queuedJobTimeout = 30 * time.Minute

// QueuedJobTimeoutMessage is the error of a job that never got a worker.
const QueuedJobTimeoutMessage = "The server was too busy to start the clip. Try again later."

// StartClipCleanUpScheduler periodically deletes clips and finished jobs that
// are older than the configured retention, and fails jobs that waited in the
// queue for too long. It stops when ctx is done.
func StartClipCleanUpScheduler(ctx context.Context) {
	cfg := config.CONFIG.ClipCleanUpSchedulerConfig
	if !cfg.IsEnabled {
		slog.Info("Clip cleanup disabled")
		return
	}

	interval := time.Duration(cfg.IntervalInMinutes) * time.Minute
	retention := time.Duration(cfg.RetentionInMinutes) * time.Minute
	slog.Info("Clip cleanup scheduled", "interval", interval, "retention", retention, "dir", cfg.ClipDirectoryPath)

	go run(ctx, interval, func(now time.Time) {
		cleanUp(now, retention, cfg.ClipDirectoryPath)
	})
}

// run calls tick every interval until ctx is done.
func run(ctx context.Context, interval time.Duration, tick func(now time.Time)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			tick(now)
		}
	}
}

// cleanUp is one cleanup pass as of now: it fails jobs that waited too long
// for a worker, then deletes clips and drops finished jobs older than
// retention.
func cleanUp(now time.Time, retention time.Duration, clipDir string) {
	for _, id := range jobs.FailQueuedBefore(now.Add(-queuedJobTimeout), QueuedJobTimeoutMessage) {
		slog.Info("Job failed: still queued", "job", id, "after", queuedJobTimeout)
	}
	cutoff := now.Add(-retention)
	cleanUpOldClips(cutoff, clipDir)
	for _, id := range jobs.RemoveFinishedBefore(cutoff) {
		slog.Info("Job removed", "job", id)
	}
}

func cleanUpOldClips(cutoff time.Time, clipDir string) {
	files, err := os.ReadDir(clipDir)
	if err != nil {
		slog.Error("Failed to read clip directory", "dir", clipDir, "err", err)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filePath := filepath.Join(clipDir, file.Name())
		info, err := file.Info()
		if err != nil {
			slog.Error("Failed to get file info", "path", filePath, "err", err)
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filePath); err != nil {
				slog.Error("Failed to delete file", "path", filePath, "err", err)
			} else {
				slog.Info("Deleted old clip", "path", filePath)
			}
		}
	}
}
