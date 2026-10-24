package jobs

import (
	"slices"
	"testing"
	"time"
)

func TestNewJob(t *testing.T) {
	job := NewJob()

	if job.Status != StatusQueued {
		t.Errorf("Expected job status to be 'queued', got %v", job.Status)
	}
	if job.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if _, exists := GetJobById(job.ID); !exists {
		t.Errorf("Job with ID %v does not exist in the store", job.ID)
	}
}

func TestStartJob(t *testing.T) {
	job := NewJob()
	StartJob(job.ID)

	got, _ := GetJobById(job.ID)
	if got.Status != StatusProcessing {
		t.Errorf("Expected job status to be 'processing', got %v", got.Status)
	}
	if got.StartedAt.IsZero() {
		t.Error("Expected StartedAt to be set")
	}
}

func TestFailJob(t *testing.T) {
	job := NewJob()
	FailJob(job.ID, "An error occurred")

	got, _ := GetJobById(job.ID)
	if got.Status != StatusError {
		t.Errorf("Expected job status to be 'error', got %v", got.Status)
	}
	if got.Error != "An error occurred" {
		t.Errorf("Unexpected error message %q", got.Error)
	}
	if got.CompletedAt.IsZero() {
		t.Error("Expected CompletedAt to be set on failure")
	}
}

func TestCompleteJob(t *testing.T) {
	job := NewJob()
	CompleteJob(job.ID, "/path/to/file.mp4", "Title [0-00-01 - 0-00-02].mp4")

	got, _ := GetJobById(job.ID)
	if got.Status != StatusCompleted {
		t.Errorf("Expected job status to be 'completed', got %v", got.Status)
	}
	if got.FilePath != "/path/to/file.mp4" {
		t.Errorf("Unexpected file path %q", got.FilePath)
	}
	if got.FileName != "Title [0-00-01 - 0-00-02].mp4" {
		t.Errorf("Unexpected file name %q", got.FileName)
	}
}

func TestGetJobByIdReturnsCopy(t *testing.T) {
	job := NewJob()
	got, _ := GetJobById(job.ID)
	got.Status = StatusError

	again, _ := GetJobById(job.ID)
	if again.Status != StatusQueued {
		t.Errorf("Mutating the returned job changed the store: %v", again.Status)
	}

	if _, exists := GetJobById("nonexistent-id"); exists {
		t.Error("Expected job with nonexistent ID to not exist")
	}
}

func TestRemoveFinishedBefore(t *testing.T) {
	done := NewJob()
	CompleteJob(done.ID, "x", "x")
	failed := NewJob()
	FailJob(failed.ID, "boom")
	running := NewJob()
	StartJob(running.ID)

	removed := RemoveFinishedBefore(time.Now().Add(time.Second))

	for _, id := range []string{done.ID, failed.ID} {
		if _, exists := GetJobById(id); exists {
			t.Errorf("Expected finished job %s to be removed", id)
		}
	}
	if _, exists := GetJobById(running.ID); !exists {
		t.Error("Expected running job to be kept")
	}
	if len(removed) < 2 {
		t.Errorf("Expected at least 2 removed IDs, got %v", removed)
	}
}

func TestStartJobOnlyStartsQueuedJobs(t *testing.T) {
	job := NewJob()
	FailJob(job.ID, "gave up")
	if StartJob(job.ID) {
		t.Error("StartJob must not start a failed job")
	}
	if got, _ := GetJobById(job.ID); got.Status != StatusError {
		t.Errorf("status = %s; want error", got.Status)
	}
	if StartJob("nonexistent-id") {
		t.Error("StartJob must not start an unknown job")
	}
	if queued := NewJob(); !StartJob(queued.ID) {
		t.Error("StartJob must start a queued job")
	}
}

func TestTryNewJobLimitsActiveJobs(t *testing.T) {
	base := ActiveCount()

	first, ok := TryNewJob(base + 1)
	if !ok || first.Status != StatusQueued {
		t.Fatalf("expected a job below the limit, got %+v %v", first, ok)
	}
	if _, ok := TryNewJob(base + 1); ok {
		t.Fatal("expected the limit to reject a second job")
	}
	if ActiveCount() != base+1 {
		t.Errorf("ActiveCount = %d; want %d", ActiveCount(), base+1)
	}

	StartJob(first.ID) // processing still counts
	if _, ok := TryNewJob(base + 1); ok {
		t.Fatal("a processing job must count towards the limit")
	}

	CompleteJob(first.ID, "x", "x") // finished jobs don't
	if _, ok := TryNewJob(base + 1); !ok {
		t.Error("expected room again once the job finished")
	}
}

func TestFailQueuedBefore(t *testing.T) {
	stale := NewJob()
	running := NewJob()
	StartJob(running.ID)

	failed := FailQueuedBefore(time.Now().Add(time.Second), "too busy")

	if got, _ := GetJobById(stale.ID); got.Status != StatusError || got.Error != "too busy" || got.CompletedAt.IsZero() {
		t.Errorf("stale queued job = %+v; want failed", got)
	}
	if got, _ := GetJobById(running.ID); got.Status != StatusProcessing {
		t.Errorf("processing job = %s; want processing", got.Status)
	}
	if !slices.Contains(failed, stale.ID) || slices.Contains(failed, running.ID) {
		t.Errorf("failed IDs = %v", failed)
	}

	fresh := NewJob()
	FailQueuedBefore(time.Now().Add(-time.Hour), "too busy")
	if got, _ := GetJobById(fresh.ID); got.Status != StatusQueued {
		t.Errorf("fresh queued job = %s; want queued", got.Status)
	}
}
