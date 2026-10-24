package jobs

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	StatusQueued     JobStatus = "queued"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusError      JobStatus = "error"
)

type Job struct {
	ID          string    `json:"id"`
	Status      JobStatus `json:"status"`
	FilePath    string    `json:"filePath,omitempty"`
	FileName    string    `json:"fileName,omitempty"` // readable download name
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt"`
}

var (
	jobs     = make(map[string]*Job)
	jobsLock sync.Mutex
)

func newJobLocked() Job {
	job := &Job{
		ID:        uuid.New().String(),
		Status:    StatusQueued,
		CreatedAt: time.Now(),
	}
	jobs[job.ID] = job
	return *job
}

func NewJob() Job {
	jobsLock.Lock()
	defer jobsLock.Unlock()
	return newJobLocked()
}

// TryNewJob creates a queued job unless maxActive jobs are already queued or
// processing. ok is false when the limit was reached.
func TryNewJob(maxActive int) (job Job, ok bool) {
	jobsLock.Lock()
	defer jobsLock.Unlock()
	if activeCountLocked() >= maxActive {
		return Job{}, false
	}
	return newJobLocked(), true
}

// ActiveCount returns how many jobs are queued or processing.
func ActiveCount() int {
	jobsLock.Lock()
	defer jobsLock.Unlock()
	return activeCountLocked()
}

func activeCountLocked() int {
	n := 0
	for _, job := range jobs {
		if job.Status == StatusQueued || job.Status == StatusProcessing {
			n++
		}
	}
	return n
}

func update(jobID string, apply func(job *Job)) {
	jobsLock.Lock()
	defer jobsLock.Unlock()
	if job, exists := jobs[jobID]; exists {
		apply(job)
	}
}

// StartJob moves a queued job to processing. It returns false if the job is
// gone or no longer queued (for example failed by FailQueuedBefore), in which
// case the caller must not process it.
func StartJob(jobID string) bool {
	started := false
	update(jobID, func(job *Job) {
		if job.Status != StatusQueued {
			return
		}
		job.Status = StatusProcessing
		job.StartedAt = time.Now()
		started = true
	})
	return started
}

func FailJob(jobID, errorMsg string) {
	update(jobID, func(job *Job) {
		job.Status = StatusError
		job.Error = errorMsg
		job.CompletedAt = time.Now()
	})
}

func CompleteJob(jobID, filePath, fileName string) {
	update(jobID, func(job *Job) {
		job.Status = StatusCompleted
		job.FilePath = filePath
		job.FileName = fileName
		job.CompletedAt = time.Now()
	})
}

// GetJobById returns a copy of the job, so callers never read fields while a
// worker goroutine is writing them.
func GetJobById(jobID string) (Job, bool) {
	jobsLock.Lock()
	defer jobsLock.Unlock()
	job, exists := jobs[jobID]
	if !exists {
		return Job{}, false
	}
	return *job, true
}

// FailQueuedBefore fails jobs that were created before cutoff and are still
// waiting for a worker, and returns their IDs.
func FailQueuedBefore(cutoff time.Time, errorMsg string) []string {
	jobsLock.Lock()
	defer jobsLock.Unlock()

	var failed []string
	now := time.Now()
	for id, job := range jobs {
		if job.Status == StatusQueued && job.CreatedAt.Before(cutoff) {
			job.Status = StatusError
			job.Error = errorMsg
			job.CompletedAt = now
			failed = append(failed, id)
		}
	}
	return failed
}

// RemoveFinishedBefore drops completed and failed jobs that finished before
// cutoff and returns their IDs.
func RemoveFinishedBefore(cutoff time.Time) []string {
	jobsLock.Lock()
	defer jobsLock.Unlock()

	var removed []string
	for id, job := range jobs {
		finished := job.Status == StatusCompleted || job.Status == StatusError
		if finished && job.CompletedAt.Before(cutoff) {
			delete(jobs, id)
			removed = append(removed, id)
		}
	}
	return removed
}
