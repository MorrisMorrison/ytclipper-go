package scheduler

import (
	"os"
	"path/filepath"
	"time"
	"ytclipper-go/config"
	"ytclipper-go/jobs"

	"github.com/MorrisMorrison/gutils/glogger"
)

func StartClipCleanUpScheduler(){
    intervalInMinutes := time.Duration(config.CONFIG.ClipCleanUpSchedulerConfig.IntervalInMinutes)*time.Minute
    retentionInMinutes := time.Duration(config.CONFIG.ClipCleanUpSchedulerConfig.IntervalInMinutes)*time.Minute

    startFileCleanUpScheduler(intervalInMinutes, retentionInMinutes, config.CONFIG.ClipCleanUpSchedulerConfig.ClipDirectoryPath)
    startJobCleanUpScheduler(intervalInMinutes, retentionInMinutes)
}

func startFileCleanUpScheduler(interval time.Duration, retention time.Duration, clipDir string) {
	glogger.Log.Infof("Start File Cleanup: Interval %f minutes - Retention %f minutes - Clip Directory %s", interval.Minutes(), retention.Minutes(), clipDir)

    ticker := time.NewTicker(interval)
    go func() {
        for range ticker.C {
            if (!config.CONFIG.ClipCleanUpSchedulerConfig.IsEnabled){
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
            if (!config.CONFIG.ClipCleanUpSchedulerConfig.IsEnabled){
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

