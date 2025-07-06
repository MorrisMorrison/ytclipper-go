package scheduler

import (
	"os"
	"path/filepath"
	"time"
	"ytclipper-go/config"
	"ytclipper-go/cookies"
	"ytclipper-go/jobs"
	"ytclipper-go/services"

	"github.com/MorrisMorrison/gutils/glogger"
)

func StartClipCleanUpScheduler() {
	intervalInMinutes := time.Duration(config.CONFIG.ClipCleanUpSchedulerConfig.IntervalInMinutes) * time.Minute
	retentionInMinutes := time.Duration(config.CONFIG.ClipCleanUpSchedulerConfig.IntervalInMinutes) * time.Minute

	startFileCleanUpScheduler(intervalInMinutes, retentionInMinutes, config.CONFIG.ClipCleanUpSchedulerConfig.ClipDirectoryPath)
	startJobCleanUpScheduler(intervalInMinutes, retentionInMinutes)
}

func StartCookieMonitorScheduler() {
	if !config.CONFIG.CookieMonitorConfig.Enabled {
		glogger.Log.Info("Cookie monitoring is disabled")
		return
	}

	intervalHours := time.Duration(config.CONFIG.CookieMonitorConfig.IntervalHours) * time.Hour

	ntfyService := services.NewNtfyService()

	cookieNotificationService := cookies.NewCookieNotificationService(ntfyService)

	cookieMonitor := cookies.NewCookieMonitor(cookieNotificationService)

	glogger.Log.Infof("Starting cookie monitor: Interval %f hours", intervalHours.Hours())

	// Send test notification if ntfy is enabled
	if config.CONFIG.NtfyConfig.Enabled {
		glogger.Log.Info("Sending test notification...")
		if err := cookieNotificationService.SendTestNotification(); err != nil {
			glogger.Log.Errorf(err, "Failed to send test notification")
		}
	}

	if err := cookieMonitor.CheckCookieHealth(); err != nil {
		glogger.Log.Errorf(err, "Initial cookie health check failed")
	}

	ticker := time.NewTicker(intervalHours)
	go func() {
		for range ticker.C {
			if !config.CONFIG.CookieMonitorConfig.Enabled {
				continue
			}

			glogger.Log.Info("Running periodic cookie health check...")
			if err := cookieMonitor.CheckCookieHealth(); err != nil {
				glogger.Log.Errorf(err, "Periodic cookie health check failed")
			}
		}
	}()
}

func startFileCleanUpScheduler(interval time.Duration, retention time.Duration, clipDir string) {
	glogger.Log.Infof("Start File Cleanup: Interval %f minutes - Retention %f minutes - Clip Directory %s", interval.Minutes(), retention.Minutes(), clipDir)

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if !config.CONFIG.ClipCleanUpSchedulerConfig.IsEnabled {
				continue
			}

			cleanUpOldClips(retention, clipDir)
		}
	}()
}

func startJobCleanUpScheduler(interval time.Duration, retention time.Duration) {
	glogger.Log.Infof("Start Job Cleanup: Retention %f minutes", retention.Minutes())

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if !config.CONFIG.ClipCleanUpSchedulerConfig.IsEnabled {
				continue
			}

			cleanUpOldJobs(retention)
		}
	}()
}

func cleanUpOldClips(retention time.Duration, clipDir string) {
	now := time.Now()

	files, err := os.ReadDir(clipDir)
	if err != nil {
		glogger.Log.Errorf(err, "Failed to read directory: %s")
		return
	}

	for _, file := range files {
		filePath := filepath.Join(clipDir, file.Name())
		info, err := file.Info()
		if err != nil {
			glogger.Log.Errorf(err, "Failed to get file info for: %s", filePath)
			continue
		}

		if now.Sub(info.ModTime()) > retention {
			if err := os.Remove(filePath); err != nil {
				glogger.Log.Errorf(err, "Failed to delete file: %s", filePath)
			} else {
				glogger.Log.Errorf(err, "File %s deleted successfully", filePath)
			}
		}
	}
}

func cleanUpOldJobs(retention time.Duration) {
	now := time.Now()
	jobs.JobsLock.Lock()
	defer jobs.JobsLock.Unlock()

	for jobID, job := range jobs.Jobs {
		if job.Status == jobs.StatusCompleted && now.Sub(job.CompletedAt) > retention {
			delete(jobs.Jobs, jobID)
			glogger.Log.Infof("Job %s removed from Jobs map", jobID)
		}
	}
}
