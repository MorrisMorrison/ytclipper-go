package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MorrisMorrison/ytclipper-go/internal/config"
	"github.com/MorrisMorrison/ytclipper-go/internal/jobs"
)

func writeFile(t *testing.T, path string, modTime time.Time) {
	t.Helper()
	if err := os.WriteFile(path, []byte("clip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestCleanUpOldClips(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	oldClip := filepath.Join(dir, "old.mp4")
	newClip := filepath.Join(dir, "new.mp4")
	subDir := filepath.Join(dir, "keep")
	writeFile(t, oldClip, now.Add(-time.Hour))
	writeFile(t, newClip, now)
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(subDir, now.Add(-time.Hour), now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	cleanUpOldClips(now.Add(-30*time.Minute), dir)

	if exists(oldClip) {
		t.Error("a clip older than the cutoff must be deleted")
	}
	if !exists(newClip) {
		t.Error("a clip newer than the cutoff must be kept")
	}
	if !exists(subDir) {
		t.Error("directories must be left alone")
	}
}

func TestCleanUpOldClipsMissingDirectory(t *testing.T) {
	// Logs and returns; must not panic.
	cleanUpOldClips(time.Now(), filepath.Join(t.TempDir(), "missing"))
}

func TestCleanUp(t *testing.T) {
	dir := t.TempDir()
	clip := filepath.Join(dir, "clip.mp4")
	writeFile(t, clip, time.Now())

	stale := jobs.NewJob() // never got a worker
	running := jobs.NewJob()
	jobs.StartJob(running.ID)
	done := jobs.NewJob()
	jobs.CompleteJob(done.ID, clip, "clip.mp4")

	// Two hours from now: the queued job has waited past queuedJobTimeout,
	// and everything finished is older than the 30-minute retention.
	cleanUp(time.Now().Add(2*time.Hour), 30*time.Minute, dir)

	if _, ok := jobs.GetJobById(stale.ID); ok {
		t.Error("a job queued for too long must be failed and then dropped")
	}
	if _, ok := jobs.GetJobById(done.ID); ok {
		t.Error("a finished job older than the retention must be dropped")
	}
	if got, ok := jobs.GetJobById(running.ID); !ok || got.Status != jobs.StatusProcessing {
		t.Errorf("a processing job must be kept, got %+v (exists %v)", got, ok)
	}
	if exists(clip) {
		t.Error("the old clip must be deleted")
	}
}

func TestCleanUpFailsStaleQueuedJob(t *testing.T) {
	stale := jobs.NewJob()

	// Past the queue timeout but within the retention: failed, not dropped.
	cleanUp(time.Now().Add(queuedJobTimeout+time.Minute), 24*time.Hour, t.TempDir())

	got, ok := jobs.GetJobById(stale.ID)
	if !ok || got.Status != jobs.StatusError || got.Error != QueuedJobTimeoutMessage {
		t.Errorf("stale queued job = %+v (exists %v); want failed with the timeout message", got, ok)
	}
}

func TestRunTicksUntilCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var ticks atomic.Int32
	stopped := make(chan struct{})
	go func() {
		run(ctx, time.Millisecond, func(time.Time) { ticks.Add(1) })
		close(stopped)
	}()

	deadline := time.After(2 * time.Second)
	for ticks.Load() < 2 {
		select {
		case <-deadline:
			t.Fatal("run did not tick")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("run did not stop after the context was cancelled")
	}
}

func TestStartClipCleanUpScheduler(t *testing.T) {
	original := config.CONFIG.ClipCleanUpSchedulerConfig
	t.Cleanup(func() { config.CONFIG.ClipCleanUpSchedulerConfig = original })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	config.CONFIG.ClipCleanUpSchedulerConfig.IsEnabled = false
	StartClipCleanUpScheduler(ctx) // disabled: returns without starting anything

	config.CONFIG.ClipCleanUpSchedulerConfig = config.ClipCleanUpSchedulerConfig{
		IsEnabled:          true,
		IntervalInMinutes:  1,
		RetentionInMinutes: 30,
		ClipDirectoryPath:  t.TempDir(),
	}
	StartClipCleanUpScheduler(ctx) // starts the loop; cancel stops it
}
