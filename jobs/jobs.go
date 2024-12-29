package jobs

import (
	"fmt"
	"sync"
	"ytclipper-go/handlers"
)

type JobStatus string

const (
    StatusQueued    JobStatus = "queued"
    StatusProcessing JobStatus = "processing"
    StatusCompleted JobStatus = "completed"
    StatusError      JobStatus = "error"
)

type Job struct {
    ID       string     `json:"id"`
    Status   JobStatus  `json:"status"`
    FilePath string     `json:"filePath,omitempty"`
    Error    string     `json:"error,omitempty"`
}

var (
    Jobs     = make(map[string]*Job) 
    JobsLock = sync.Mutex{}
    JobQueue = make(chan string, 100)    
)

func StartWorkerPool(workerCount int) {
    for i := 0; i < workerCount; i++ {
        go worker(i)
    }
}

func worker(workerID int) {
    for jobID := range JobQueue {
        JobsLock.Lock()
        job, exists := Jobs[jobID]
        JobsLock.Unlock()

        if !exists {
            fmt.Printf("Worker %d: Job %s not found\n", workerID, jobID)
            continue
        }

        handlers.ProcessClip(jobId, )
    }
}
