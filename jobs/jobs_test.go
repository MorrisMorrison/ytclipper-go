package jobs

import (
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
	CompleteJob(job.ID, "/path/to/file.mp4")

	got, _ := GetJobById(job.ID)
	if got.Status != StatusCompleted {
		t.Errorf("Expected job status to be 'completed', got %v", got.Status)
	}
	if got.FilePath != "/path/to/file.mp4" {
		t.Errorf("Unexpected file path %q", got.FilePath)
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
	CompleteJob(done.ID, "x")
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
