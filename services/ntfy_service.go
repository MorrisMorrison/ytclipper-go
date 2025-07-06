package services

import (
	"bytes"
	"fmt"
	"net/http"
	"time"
	"ytclipper-go/config"

	"github.com/MorrisMorrison/gutils/glogger"
)

type NtfyService struct {
	serverURL string
	enabled   bool
}

type NotificationRequest struct {
	Topic    string
	Title    string
	Message  string
	Priority string
	Tags     []string
}

const (
	PriorityMin     = "min"
	PriorityLow     = "low"
	PriorityDefault = "default"
	PriorityHigh    = "high"
	PriorityMax     = "max"
)

func NewNtfyService() *NtfyService {
	return &NtfyService{
		serverURL: config.CONFIG.NtfyConfig.ServerURL,
		enabled:   config.CONFIG.NtfyConfig.Enabled,
	}
}

func (ns *NtfyService) SendNotification(req NotificationRequest) error {
	if !ns.enabled {
		glogger.Log.Infof("Ntfy notifications disabled - skipping notification: %s", req.Title)
		return nil
	}

	if ns.serverURL == "" {
		return fmt.Errorf("ntfy server URL not configured")
	}

	if req.Topic == "" {
		return fmt.Errorf("notification topic is required")
	}

	url := fmt.Sprintf("%s/%s", ns.serverURL, req.Topic)

	httpReq, err := http.NewRequest("POST", url, bytes.NewBufferString(req.Message))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Title", req.Title)
	httpReq.Header.Set("User-Agent", "ytclipper-go/1.0")

	if req.Priority != "" {
		httpReq.Header.Set("Priority", req.Priority)
	}

	if len(req.Tags) > 0 {
		tagsStr := ""
		for i, tag := range req.Tags {
			if i > 0 {
				tagsStr += ","
			}
			tagsStr += tag
		}
		httpReq.Header.Set("Tags", tagsStr)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	glogger.Log.Infof("Sending ntfy notification to topic '%s': %s", req.Topic, req.Title)

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy server returned status %d", resp.StatusCode)
	}

	glogger.Log.Infof("Successfully sent ntfy notification to topic '%s': %s", req.Topic, req.Title)
	return nil
}

func (ns *NtfyService) SendSimpleNotification(topic, title, message string) error {
	return ns.SendNotification(NotificationRequest{
		Topic:    topic,
		Title:    title,
		Message:  message,
		Priority: PriorityDefault,
		Tags:     []string{"ytclipper"},
	})
}

func (ns *NtfyService) SendAlertNotification(topic, title, message string, tags []string) error {
	return ns.SendNotification(NotificationRequest{
		Topic:    topic,
		Title:    title,
		Message:  message,
		Priority: PriorityHigh,
		Tags:     append(tags, "alert", "ytclipper"),
	})
}

func (ns *NtfyService) SendCriticalNotification(topic, title, message string, tags []string) error {
	return ns.SendNotification(NotificationRequest{
		Topic:    topic,
		Title:    title,
		Message:  message,
		Priority: PriorityMax,
		Tags:     append(tags, "critical", "ytclipper"),
	})
}

func (ns *NtfyService) TestNotification(topic string) error {
	if !ns.enabled {
		return fmt.Errorf("ntfy notifications are disabled")
	}

	return ns.SendNotification(NotificationRequest{
		Topic:    topic,
		Title:    "🧪 ytclipper-go Test",
		Message:  "Notification system is working correctly!\n\nThis is a test notification to verify your ntfy configuration.",
		Priority: PriorityLow,
		Tags:     []string{"test", "ytclipper"},
	})
}

func (ns *NtfyService) IsEnabled() bool {
	return ns.enabled
}

func (ns *NtfyService) GetServerURL() string {
	return ns.serverURL
}
