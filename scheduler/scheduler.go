package scheduler

import (
	"os"
	"path/filepath"
	"time"
	"ytclipper-go/config"
	"ytclipper-go/jobs"

	"github.com/MorrisMorrison/gutils/glogger"
)

// queuedJobTimeout is how long a job may wait for a worker before cleanup
// fails it.
const queuedJobTimeout = 30 * time.Minute

// QueuedJobTimeoutMessage is the error of a job that never got a worker.
const QueuedJobTimeoutMessage = "The server was too busy to start the clip. Try again later."

// StartClipCleanUpScheduler periodically deletes clips and finished jobs that
// are older than the configured retention, and fails jobs that waited in the
// queue for too long.
func StartClipCleanUpScheduler() {
	cfg := config.CONFIG.ClipCleanUpSchedulerConfig
	if !cfg.IsEnabled {
		glogger.Log.Info("Clip cleanup disabled")
		return
	}

	interval := time.Duration(cfg.IntervalInMinutes) * time.Minute
	retention := time.Duration(cfg.RetentionInMinutes) * time.Minute
	glogger.Log.Infof("Clip cleanup: every %v, retention %v, directory %s", interval, retention, cfg.ClipDirectoryPath)

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			for _, id := range jobs.FailQueuedBefore(time.Now().Add(-queuedJobTimeout), QueuedJobTimeoutMessage) {
				glogger.Log.Infof("Job %s failed: still queued after %v", id, queuedJobTimeout)
			}
			cutoff := time.Now().Add(-retention)
			cleanUpOldClips(cutoff, cfg.ClipDirectoryPath)
			for _, id := range jobs.RemoveFinishedBefore(cutoff) {
				glogger.Log.Infof("Job %s removed", id)
			}
		}
	}()
}

func cleanUpOldClips(cutoff time.Time, clipDir string) {
	files, err := os.ReadDir(clipDir)
	if err != nil {
		glogger.Log.Errorf(err, "Failed to read clip directory: %s", clipDir)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filePath := filepath.Join(clipDir, file.Name())
		info, err := file.Info()
		if err != nil {
			glogger.Log.Errorf(err, "Failed to get file info for: %s", filePath)
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filePath); err != nil {
				glogger.Log.Errorf(err, "Failed to delete file: %s", filePath)
			} else {
				glogger.Log.Infof("Deleted old clip %s", filePath)
			}
		}
	}
}
