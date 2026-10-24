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
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt"`
}

var (
	jobs     = make(map[string]*Job)
	jobsLock sync.Mutex
)

func NewJob() Job {
	job := &Job{
		ID:        uuid.New().String(),
		Status:    StatusQueued,
		CreatedAt: time.Now(),
	}
	jobsLock.Lock()
	jobs[job.ID] = job
	jobsLock.Unlock()

	return *job
}

func update(jobID string, apply func(job *Job)) {
	jobsLock.Lock()
	defer jobsLock.Unlock()
	if job, exists := jobs[jobID]; exists {
		apply(job)
	}
}

func StartJob(jobID string) {
	update(jobID, func(job *Job) {
		job.Status = StatusProcessing
		job.StartedAt = time.Now()
	})
}

func FailJob(jobID, errorMsg string) {
	update(jobID, func(job *Job) {
		job.Status = StatusError
		job.Error = errorMsg
		job.CompletedAt = time.Now()
	})
}

func CompleteJob(jobID, filePath string) {
	update(jobID, func(job *Job) {
		job.Status = StatusCompleted
		job.FilePath = filePath
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
