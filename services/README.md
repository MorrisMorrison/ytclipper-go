# Services Package

This package contains reusable services that can be used throughout the ytclipper-go application.

## NtfyService

A generic notification service that can send notifications to any ntfy server and topic.

### Usage Examples

#### Basic Notification
```go
import "ytclipper-go/services"

ntfyService := services.NewNtfyService()
err := ntfyService.SendSimpleNotification("my-topic", "Hello", "This is a test message")
```

#### Custom Notification
```go
err := ntfyService.SendNotification(services.NotificationRequest{
    Topic:    "alerts",
    Title:    "System Alert",
    Message:  "Something important happened",
    Priority: services.PriorityHigh,
    Tags:     []string{"system", "alert", "production"},
})
```

#### Alert Notifications
```go
// High priority alert
err := ntfyService.SendAlertNotification(
    "alerts", 
    "Service Warning", 
    "High memory usage detected",
    []string{"memory", "performance"},
)

// Critical notification
err := ntfyService.SendCriticalNotification(
    "critical", 
    "Service Down", 
    "Database connection failed",
    []string{"database", "outage"},
)
```

#### Test Configuration
```go
err := ntfyService.TestNotification("my-topic")
```

### Configuration

The NtfyService uses global configuration:

```bash
YTCLIPPER_NTFY_ENABLED=true
YTCLIPPER_NTFY_SERVER_URL="https://ntfy.sh"
```

### Priority Levels

- `PriorityMin`: Lowest priority
- `PriorityLow`: Low priority  
- `PriorityDefault`: Normal priority
- `PriorityHigh`: High priority
- `PriorityMax`: Highest priority

### Integration Examples

#### Job Processing Notifications
```go
func ProcessJob(jobID string) {
    ntfyService := services.NewNtfyService()
    
    // Job started
    ntfyService.SendSimpleNotification("jobs", "Job Started", fmt.Sprintf("Processing job %s", jobID))
    
    // Job completed
    if err := doWork(); err != nil {
        ntfyService.SendCriticalNotification("jobs", "Job Failed", 
            fmt.Sprintf("Job %s failed: %v", jobID, err), []string{"jobs", "error"})
    } else {
        ntfyService.SendSimpleNotification("jobs", "Job Completed", 
            fmt.Sprintf("Job %s completed successfully", jobID))
    }
}
```

#### System Health Monitoring
```go
func MonitorSystemHealth() {
    ntfyService := services.NewNtfyService()
    
    if memoryUsage > 90 {
        ntfyService.SendAlertNotification("health", "High Memory Usage", 
            fmt.Sprintf("Memory usage at %d%%", memoryUsage), []string{"memory", "health"})
    }
    
    if diskSpace < 10 {
        ntfyService.SendCriticalNotification("health", "Low Disk Space", 
            fmt.Sprintf("Only %dGB remaining", diskSpace), []string{"disk", "storage"})
    }
}
```

#### Error Reporting
```go
func HandleError(err error, context string) {
    ntfyService := services.NewNtfyService()
    
    ntfyService.SendNotification(services.NotificationRequest{
        Topic:    "errors",
        Title:    "Application Error",
        Message:  fmt.Sprintf("Error in %s: %v", context, err),
        Priority: services.PriorityHigh,
        Tags:     []string{"error", "application", context},
    })
}
```

This generic design allows the NtfyService to be used for any notification needs throughout the application, not just cookie monitoring.