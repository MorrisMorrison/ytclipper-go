package cookies

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"ytclipper-go/config"
	"ytclipper-go/videoprocessing"

	"github.com/MorrisMorrison/gutils/glogger"
)

type CookieMonitor struct {
	notificationService *CookieNotificationService
}

type CookieInfo struct {
	Name      string
	ExpiresAt time.Time
	IsValid   bool
}

func NewCookieMonitor(notificationService *CookieNotificationService) *CookieMonitor {
	return &CookieMonitor{
		notificationService: notificationService,
	}
}

// ParseCookieExpiration extracts the expiration time from cookie content
func (cm *CookieMonitor) ParseCookieExpiration(cookieContent string) (*CookieInfo, error) {
	if cookieContent == "" {
		return nil, fmt.Errorf("cookie content is empty")
	}

	lines := strings.Split(cookieContent, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse Netscape cookie format: domain	flag	path	secure	expiration	name	value
		parts := strings.Split(line, "\t")
		if len(parts) < 7 {
			continue
		}

		cookieName := parts[5]
		expirationStr := parts[4]

		// Look for VISITOR_INFO1_LIVE cookie specifically
		if strings.Contains(cookieName, "VISITOR_INFO1_LIVE") {
			if expirationTimestamp, err := strconv.ParseInt(expirationStr, 10, 64); err == nil {
				expiresAt := time.Unix(expirationTimestamp, 0)

				return &CookieInfo{
					Name:      cookieName,
					ExpiresAt: expiresAt,
					IsValid:   time.Now().Before(expiresAt),
				}, nil
			}
		}
	}

	// If no VISITOR_INFO1_LIVE found, estimate 6 months from now
	glogger.Log.Warning("VISITOR_INFO1_LIVE cookie not found, estimating 6 months expiration")
	return &CookieInfo{
		Name:      "ESTIMATED",
		ExpiresAt: time.Now().Add(6 * 30 * 24 * time.Hour), // 6 months
		IsValid:   true,
	}, nil
}

// CheckCookieHealth analyzes current cookie health status
func (cm *CookieMonitor) CheckCookieHealth() error {
	glogger.Log.Info("Checking cookie health...")

	// Get cookie content from config
	cookieContent := config.CONFIG.YtDlpConfig.CookiesContent
	cookieFile := config.CONFIG.YtDlpConfig.CookiesFile

	if cookieContent == "" && cookieFile == "" {
		glogger.Log.Info("No cookies configured - using cookie-free fallback strategy")
		return nil
	}

	var cookieInfo *CookieInfo
	var err error

	if cookieContent != "" {
		cookieInfo, err = cm.ParseCookieExpiration(cookieContent)
	} else if cookieFile != "" {
		// Read cookie file content
		content, readErr := cm.readCookieFile(cookieFile)
		if readErr != nil {
			return fmt.Errorf("failed to read cookie file %s: %w", cookieFile, readErr)
		}
		cookieInfo, err = cm.ParseCookieExpiration(content)
	}

	if err != nil {
		return fmt.Errorf("failed to parse cookie expiration: %w", err)
	}

	// Check expiration status
	timeUntilExpiry := time.Until(cookieInfo.ExpiresAt)

	glogger.Log.Infof("Cookie %s expires at %s (in %s)",
		cookieInfo.Name,
		cookieInfo.ExpiresAt.Format(time.RFC3339),
		timeUntilExpiry.Round(time.Hour))

	// Perform API validation if enabled
	var apiValidationError error
	if config.CONFIG.CookieMonitorConfig.APIValidationEnabled {
		glogger.Log.Info("Performing API validation for cookie health...")
		apiValidationError = cm.ValidateCookieWithAPI(config.CONFIG.CookieMonitorConfig)
		if apiValidationError != nil {
			glogger.Log.Errorf(apiValidationError, "API validation failed")
			// Send API validation failure notification
			if notifyErr := cm.notificationService.SendAPIValidationFailedNotification(apiValidationError); notifyErr != nil {
				glogger.Log.Errorf(notifyErr, "Failed to send API validation failure notification")
			}
		} else {
			glogger.Log.Info("API validation successful - cookies are working properly")
		}
	}

	// Send notifications based on time remaining (only if API validation passed or is disabled)
	if apiValidationError == nil {
		if err := cm.evaluateAndNotify(cookieInfo, timeUntilExpiry); err != nil {
			glogger.Log.Errorf(err, "Failed to send notification")
		}
	} else {
		// API validation failed - this is more critical than time-based warnings
		glogger.Log.Warning("Skipping time-based notifications due to API validation failure")
	}

	return nil
}

// evaluateAndNotify sends appropriate notifications based on cookie expiration
func (cm *CookieMonitor) evaluateAndNotify(cookieInfo *CookieInfo, timeUntilExpiry time.Duration) error {
	warningThreshold := time.Duration(config.CONFIG.CookieMonitorConfig.WarningThresholdDays) * 24 * time.Hour
	urgentThreshold := time.Duration(config.CONFIG.CookieMonitorConfig.UrgentThresholdDays) * 24 * time.Hour

	switch {
	case timeUntilExpiry <= 0:
		return cm.notificationService.SendExpiredNotification(cookieInfo.Name, cookieInfo.ExpiresAt)
	case timeUntilExpiry <= urgentThreshold:
		return cm.notificationService.SendUrgentNotification(cookieInfo.Name, timeUntilExpiry, cookieInfo.ExpiresAt)
	case timeUntilExpiry <= warningThreshold:
		return cm.notificationService.SendWarningNotification(cookieInfo.Name, timeUntilExpiry, cookieInfo.ExpiresAt)
	default:
		glogger.Log.Infof("Cookie is healthy (expires in %s)", timeUntilExpiry.Round(time.Hour))
		return nil
	}
}

// readCookieFile reads cookie content from file
func (cm *CookieMonitor) readCookieFile(cookieFilePath string) (string, error) {
	content, err := os.ReadFile(cookieFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read cookie file: %w", err)
	}
	return string(content), nil
}

// ValidateCookieWithAPI tests cookie functionality by attempting to get video duration
func (cm *CookieMonitor) ValidateCookieWithAPI(config config.CookieMonitorConfig) error {
	if config.TestVideoURL == "" {
		return fmt.Errorf("test video URL is not configured")
	}

	glogger.Log.Infof("Validating cookie with API using test video: %s", config.TestVideoURL)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.APIValidationTimeoutSecs)*time.Second)
	defer cancel()

	// Channel to receive the result
	resultChan := make(chan error, 1)

	// Run the API call in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- fmt.Errorf("API validation panicked: %v", r)
			}
		}()

		duration, err := videoprocessing.GetVideoDuration(config.TestVideoURL)
		if err != nil {
			resultChan <- fmt.Errorf("failed to get video duration: %w", err)
			return
		}

		if duration == "" {
			resultChan <- fmt.Errorf("received empty duration response")
			return
		}

		glogger.Log.Infof("Cookie validation successful - retrieved duration: %s", duration)
		resultChan <- nil
	}()

	// Wait for result or timeout
	select {
	case err := <-resultChan:
		if err != nil {
			return fmt.Errorf("API validation failed: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("API validation timed out after %d seconds", config.APIValidationTimeoutSecs)
	}
}
