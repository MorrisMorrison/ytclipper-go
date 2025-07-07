package cookies

import (
	"fmt"
	"time"
	"ytclipper-go/config"
	"ytclipper-go/services"

	"github.com/MorrisMorrison/gutils/glogger"
)

type CookieNotificationService struct {
	ntfyService *services.NtfyService
	topic       string
}

func NewCookieNotificationService(ntfyService *services.NtfyService) *CookieNotificationService {
	return &CookieNotificationService{
		ntfyService: ntfyService,
		topic:       config.CONFIG.CookieMonitorConfig.NtfyTopic,
	}
}

func (cns *CookieNotificationService) SendWarningNotification(cookieName string, timeUntilExpiry time.Duration, expiresAt time.Time) error {
	if !cns.ntfyService.IsEnabled() {
		glogger.Log.Info("Ntfy notifications disabled - skipping warning notification")
		return nil
	}

	days := int(timeUntilExpiry.Hours() / 24)
	title := "🟡 YouTube Cookie Warning"
	message := fmt.Sprintf("YouTube cookie '%s' will expire in %d days\n\nExpiration: %s\n\nPlease update your cookies soon to avoid disruption.",
		cookieName,
		days,
		expiresAt.Format("2006-01-02 15:04:05 MST"))

	return cns.ntfyService.SendAlertNotification(
		cns.topic,
		title,
		message,
		[]string{"cookie", "youtube", "warning"},
	)
}

func (cns *CookieNotificationService) SendUrgentNotification(cookieName string, timeUntilExpiry time.Duration, expiresAt time.Time) error {
	if !cns.ntfyService.IsEnabled() {
		glogger.Log.Info("Ntfy notifications disabled - skipping urgent notification")
		return nil
	}

	hours := int(timeUntilExpiry.Hours())
	title := "🔴 URGENT: YouTube Cookie Expiring"
	message := fmt.Sprintf("⚠️ YouTube cookie '%s' expires in %d hours!\n\nExpiration: %s\n\n🚨 ACTION REQUIRED: Update cookies immediately to prevent service disruption.",
		cookieName,
		hours,
		expiresAt.Format("2006-01-02 15:04:05 MST"))

	return cns.ntfyService.SendCriticalNotification(
		cns.topic,
		title,
		message,
		[]string{"cookie", "youtube", "urgent"},
	)
}

func (cns *CookieNotificationService) SendExpiredNotification(cookieName string, expiredAt time.Time) error {
	if !cns.ntfyService.IsEnabled() {
		glogger.Log.Info("Ntfy notifications disabled - skipping expired notification")
		return nil
	}

	title := "❌ YouTube Cookie EXPIRED"
	message := fmt.Sprintf("💥 YouTube cookie '%s' has EXPIRED!\n\nExpired: %s\n\n🛠️ SERVICE DISRUPTION: ytclipper-go is now using fallback strategy. Update cookies to restore full functionality.",
		cookieName,
		expiredAt.Format("2006-01-02 15:04:05 MST"))

	return cns.ntfyService.SendCriticalNotification(
		cns.topic,
		title,
		message,
		[]string{"cookie", "youtube", "expired"},
	)
}

func (cns *CookieNotificationService) SendHealthyNotification(cookieName string, timeUntilExpiry time.Duration) error {
	if !cns.ntfyService.IsEnabled() {
		glogger.Log.Info("Ntfy notifications disabled - skipping healthy notification")
		return nil
	}

	days := int(timeUntilExpiry.Hours() / 24)
	title := "✅ YouTube Cookie Healthy"
	message := fmt.Sprintf("YouTube cookie '%s' is healthy\n\nExpires in %d days\n\nNo action required.",
		cookieName,
		days)

	return cns.ntfyService.SendNotification(services.NotificationRequest{
		Topic:    cns.topic,
		Title:    title,
		Message:  message,
		Priority: services.PriorityLow,
		Tags:     []string{"cookie", "youtube", "healthy"},
	})
}

func (cns *CookieNotificationService) SendAPIValidationFailedNotification(validationError error) error {
	if !cns.ntfyService.IsEnabled() {
		glogger.Log.Info("Ntfy notifications disabled - skipping API validation failed notification")
		return nil
	}

	title := "🔴 Cookie API Validation Failed"
	message := fmt.Sprintf("⚠️ YouTube cookie API validation failed!\n\nError: %s\n\n🚨 This indicates cookies may be invalid or expired, even if they haven't reached their expiration date.\n\n🛠️ ACTION REQUIRED: Check and update your YouTube cookies immediately.",
		validationError.Error())

	return cns.ntfyService.SendCriticalNotification(
		cns.topic,
		title,
		message,
		[]string{"cookie", "youtube", "api-validation", "failed"},
	)
}

func (cns *CookieNotificationService) SendTestNotification() error {
	if !cns.ntfyService.IsEnabled() {
		return fmt.Errorf("ntfy notifications are disabled")
	}

	title := "🧪 Cookie Monitoring Test"
	message := "Cookie monitoring is working correctly!\n\nThis is a test notification to verify your cookie monitoring configuration."

	return cns.ntfyService.SendNotification(services.NotificationRequest{
		Topic:    cns.topic,
		Title:    title,
		Message:  message,
		Priority: services.PriorityLow,
		Tags:     []string{"cookie", "test", "monitoring"},
	})
}
