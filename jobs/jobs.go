package jobs

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
    StatusQueued        JobStatus = "queued"
    StatusProcessing    JobStatus = "processing"
    StatusCompleted     JobStatus = "completed"
    StatusError         JobStatus = "error"
)

type Job struct {
    ID           string         `json:"id"`
    Status       JobStatus      `json:"status"`
    FilePath     string         `json:"filePath,omitempty"`
    Error        string         `json:"error,omitempty"`
    StartedAt    time.Time      `json:"startedAt"`
    CompletedAt  time.Time      `json:"completedAt"`
}

var (
    Jobs     = make(map[string]*Job) 
    JobsLock = sync.Mutex{}
)

func NewJob() *Job{
    jobID := uuid.New().String()
    job := &Job{
        ID:     jobID,
        Status: StatusQueued,
    }
    JobsLock.Lock()
    Jobs[job.ID] = job
    JobsLock.Unlock()

    return job
}

func UpdateJobStatus(jobID string, status JobStatus){
    JobsLock.Lock()
    defer JobsLock.Unlock()
    job, exists := Jobs[jobID]
    if exists {
        job.Status = status
    }
}

func FailJob(jobID, errorMsg string) {
    JobsLock.Lock()
    defer JobsLock.Unlock()
    job, exists := Jobs[jobID]
    if exists {
        job.Status = StatusError
        job.Error = errorMsg
    }
}

func CompleteJob(jobID, filePath string) {
    JobsLock.Lock()
    defer JobsLock.Unlock()
    job, exists := Jobs[jobID]
    if exists {
        job.Status = StatusCompleted
        job.FilePath = filePath
        job.CompletedAt = time.Now()
    }
}

func StartJob(jobID string){
    JobsLock.Lock()
    defer JobsLock.Unlock()
    job, exists := Jobs[jobID]
    if exists {
        job.Status = StatusProcessing
        job.StartedAt = time.Now()
    }
}

func GetJobById(jobId string) (*Job, bool) {
    JobsLock.Lock()
    job, exists := Jobs[jobId]
    JobsLock.Unlock()

    return job, exists;
}