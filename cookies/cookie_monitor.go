package cookies

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"ytclipper-go/config"

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

	// Send notifications based on time remaining
	if err := cm.evaluateAndNotify(cookieInfo, timeUntilExpiry); err != nil {
		glogger.Log.Errorf(err, "Failed to send notification")
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