package jobs

import (
	"testing"
)

func TestNewJob(t *testing.T) {
	job := NewJob()

	if job == nil {
		t.Fatal("NewJob() returned nil")
	}

	if job.Status != StatusQueued {
		t.Errorf("Expected job status to be 'queued', got %v", job.Status)
	}

	if _, exists := Jobs[job.ID]; !exists {
		t.Errorf("Job with ID %v does not exist in Jobs map", job.ID)
	}
}

func TestUpdateJobStatus(t *testing.T) {
	job := NewJob()
	UpdateJobStatus(job.ID, StatusProcessing)

	if job.Status != StatusProcessing {
		t.Errorf("Expected job status to be 'processing', got %v", job.Status)
	}
}

func TestFailJob(t *testing.T) {
	job := NewJob()
	errorMsg := "An error occurred"
	FailJob(job.ID, errorMsg)

	if job.Status != StatusError {
		t.Errorf("Expected job status to be 'error', got %v", job.Status)
	}

	if job.Error != errorMsg {
		t.Errorf("Expected error message to be '%v', got '%v'", errorMsg, job.Error)
	}
}

func TestCompleteJob(t *testing.T) {
	job := NewJob()
	filePath := "/path/to/file.mp4"
	CompleteJob(job.ID, filePath)

	if job.Status != StatusCompleted {
		t.Errorf("Expected job status to be 'completed', got %v", job.Status)
	}

	if job.FilePath != filePath {
		t.Errorf("Expected file path to be '%v', got '%v'", filePath, job.FilePath)
	}
}

func TestStartJob(t *testing.T) {
	job := NewJob()
	StartJob(job.ID)

	if job.Status != StatusProcessing {
		t.Errorf("Expected job status to be 'processing', got %v", job.Status)
	}
}

func TestGetJobById(t *testing.T) {
	job := NewJob()
	foundJob, exists := GetJobById(job.ID)

	if !exists {
		t.Errorf("Expected job with ID %v to exist", job.ID)
	}

	if foundJob.ID != job.ID {
		t.Errorf("Expected job ID to be '%v', got '%v'", job.ID, foundJob.ID)
	}

	_, exists = GetJobById("nonexistent-id")
	if exists {
		t.Errorf("Expected job with nonexistent ID to not exist")
	}
}
