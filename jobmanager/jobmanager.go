package jobmanager

import (
	"sync"
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
)