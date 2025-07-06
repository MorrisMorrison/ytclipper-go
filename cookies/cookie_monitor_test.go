package cookies

import (
	"testing"
	"time"
	"ytclipper-go/services"
)

func TestParseCookieExpiration(t *testing.T) {
	// Mock services for testing
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)
	monitor := NewCookieMonitor(cookieNotificationService)

	// Test cookie content with VISITOR_INFO1_LIVE
	testCookieContent := `# Netscape HTTP Cookie File
.youtube.com	TRUE	/	FALSE	1704067200	VISITOR_INFO1_LIVE	xyz123abc
.youtube.com	TRUE	/	FALSE	1704067200	YSC	def456ghi`

	cookieInfo, err := monitor.ParseCookieExpiration(testCookieContent)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cookieInfo == nil {
		t.Fatal("Expected cookie info, got nil")
	}

	expectedTime := time.Unix(1704067200, 0)
	if !cookieInfo.ExpiresAt.Equal(expectedTime) {
		t.Errorf("Expected expiration time %v, got %v", expectedTime, cookieInfo.ExpiresAt)
	}

	if cookieInfo.Name != "VISITOR_INFO1_LIVE" {
		t.Errorf("Expected cookie name 'VISITOR_INFO1_LIVE', got '%s'", cookieInfo.Name)
	}
}

func TestParseCookieExpirationEmpty(t *testing.T) {
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)
	monitor := NewCookieMonitor(cookieNotificationService)

	_, err := monitor.ParseCookieExpiration("")
	if err == nil {
		t.Fatal("Expected error for empty cookie content, got nil")
	}
}

func TestParseCookieExpirationNoVisitorInfo(t *testing.T) {
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)
	monitor := NewCookieMonitor(cookieNotificationService)

	// Test cookie content without VISITOR_INFO1_LIVE
	testCookieContent := `# Netscape HTTP Cookie File
.youtube.com	TRUE	/	FALSE	1704067200	YSC	def456ghi
.youtube.com	TRUE	/	FALSE	1704067200	OTHER_COOKIE	jkl789mno`

	cookieInfo, err := monitor.ParseCookieExpiration(testCookieContent)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cookieInfo.Name != "ESTIMATED" {
		t.Errorf("Expected cookie name 'ESTIMATED', got '%s'", cookieInfo.Name)
	}

	// Should estimate 6 months from now
	estimatedTime := time.Now().Add(6 * 30 * 24 * time.Hour)
	timeDiff := cookieInfo.ExpiresAt.Sub(estimatedTime)
	if timeDiff > time.Hour || timeDiff < -time.Hour {
		t.Errorf("Estimated time too far from expected: diff = %v", timeDiff)
	}
}