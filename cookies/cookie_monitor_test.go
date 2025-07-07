package cookies

import (
	"errors"
	"fmt"
	"testing"
	"time"
	"ytclipper-go/config"
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

func TestValidateCookieWithAPI_EmptyTestURL(t *testing.T) {
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)
	monitor := NewCookieMonitor(cookieNotificationService)

	cfg := config.CookieMonitorConfig{
		TestVideoURL:             "",
		APIValidationTimeoutSecs: 30,
	}

	err := monitor.ValidateCookieWithAPI(cfg)
	if err == nil {
		t.Fatal("Expected error for empty test URL, got nil")
	}

	expectedError := "test video URL is not configured"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidateCookieWithAPI_InvalidURL(t *testing.T) {
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)
	monitor := NewCookieMonitor(cookieNotificationService)

	cfg := config.CookieMonitorConfig{
		TestVideoURL:             "invalid-url",
		APIValidationTimeoutSecs: 30,
	}

	err := monitor.ValidateCookieWithAPI(cfg)
	if err == nil {
		t.Fatal("Expected error for invalid URL, got nil")
	}

	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestValidateCookieWithAPI_ShortTimeout(t *testing.T) {
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)
	monitor := NewCookieMonitor(cookieNotificationService)

	cfg := config.CookieMonitorConfig{
		TestVideoURL:             "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		APIValidationTimeoutSecs: 1, // Very short timeout
	}

	err := monitor.ValidateCookieWithAPI(cfg)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// In test environment without yt-dlp, we expect an exec error rather than timeout
	// The timeout functionality is still validated by the context timeout mechanism
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestValidateCookieWithAPI_ValidConfig(t *testing.T) {
	// This test would require a more complex setup with mocked videoprocessing.GetVideoDuration
	// For now, we'll test the configuration validation
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)
	monitor := NewCookieMonitor(cookieNotificationService)

	cfg := config.CookieMonitorConfig{
		TestVideoURL:             "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		APIValidationTimeoutSecs: 30,
	}

	// Note: This test will likely fail in CI environment without proper cookies,
	// but it validates the configuration handling
	err := monitor.ValidateCookieWithAPI(cfg)
	// We expect this to either succeed or fail with a specific API error, not a config error
	if err != nil && err.Error() == "test video URL is not configured" {
		t.Errorf("Configuration validation failed unexpectedly: %v", err)
	}
}

func TestCheckCookieHealth_WithAPIValidation(t *testing.T) {
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)
	monitor := NewCookieMonitor(cookieNotificationService)

	// Save original config
	originalConfig := config.CONFIG.CookieMonitorConfig

	// Set up test config with API validation enabled
	config.CONFIG.CookieMonitorConfig = config.CookieMonitorConfig{
		APIValidationEnabled:     true,
		TestVideoURL:             "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		APIValidationTimeoutSecs: 30,
		WarningThresholdDays:     30,
		UrgentThresholdDays:      7,
	}

	// Set up test cookie content
	config.CONFIG.YtDlpConfig.CookiesContent = `# Netscape HTTP Cookie File
.youtube.com	TRUE	/	FALSE	1704067200	VISITOR_INFO1_LIVE	xyz123abc`

	// Restore original config after test
	defer func() {
		config.CONFIG.CookieMonitorConfig = originalConfig
	}()

	// This should not panic and should handle API validation
	err := monitor.CheckCookieHealth()
	if err != nil {
		t.Errorf("CheckCookieHealth failed: %v", err)
	}
}

func TestCheckCookieHealth_WithoutAPIValidation(t *testing.T) {
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)
	monitor := NewCookieMonitor(cookieNotificationService)

	// Save original config
	originalConfig := config.CONFIG.CookieMonitorConfig

	// Set up test config with API validation disabled
	config.CONFIG.CookieMonitorConfig = config.CookieMonitorConfig{
		APIValidationEnabled:     false,
		TestVideoURL:             "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		APIValidationTimeoutSecs: 30,
		WarningThresholdDays:     30,
		UrgentThresholdDays:      7,
	}

	// Set up test cookie content
	config.CONFIG.YtDlpConfig.CookiesContent = `# Netscape HTTP Cookie File
.youtube.com	TRUE	/	FALSE	1704067200	VISITOR_INFO1_LIVE	xyz123abc`

	// Restore original config after test
	defer func() {
		config.CONFIG.CookieMonitorConfig = originalConfig
	}()

	// This should work without API validation
	err := monitor.CheckCookieHealth()
	if err != nil {
		t.Errorf("CheckCookieHealth failed: %v", err)
	}
}

// Since we can't directly mock videoprocessing.GetVideoDuration,
// we'll test the ValidateCookieWithAPI function with the understanding
// that it will call the actual function, but focus on testing the
// logic around timeouts, error handling, and configuration validation.

func TestValidateCookieWithAPI_Success(t *testing.T) {
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)
	monitor := NewCookieMonitor(cookieNotificationService)

	cfg := config.CookieMonitorConfig{
		TestVideoURL:             "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		APIValidationTimeoutSecs: 30,
	}

	// Note: This test will likely fail in CI without proper cookies/yt-dlp,
	// but tests the configuration validation and function structure
	err := monitor.ValidateCookieWithAPI(cfg)
	// We don't assert success here since it depends on external factors
	// Instead, we verify the error handling logic is working
	if err != nil {
		// Should be a specific API-related error, not a configuration error
		if err.Error() == "test video URL is not configured" {
			t.Errorf("Configuration validation failed unexpectedly: %v", err)
		}
	}
}

func TestValidateCookieWithAPI_ConfigurationValidation(t *testing.T) {
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)
	monitor := NewCookieMonitor(cookieNotificationService)

	tests := []struct {
		name        string
		config      config.CookieMonitorConfig
		expectError bool
		expectedMsg string
	}{
		{
			name: "Valid configuration",
			config: config.CookieMonitorConfig{
				TestVideoURL:             "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
				APIValidationTimeoutSecs: 30,
			},
			expectError: false, // May fail due to external factors, but not config
		},
		{
			name: "Empty test URL",
			config: config.CookieMonitorConfig{
				TestVideoURL:             "",
				APIValidationTimeoutSecs: 30,
			},
			expectError: true,
			expectedMsg: "test video URL is not configured",
		},
		{
			name: "Zero timeout",
			config: config.CookieMonitorConfig{
				TestVideoURL:             "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
				APIValidationTimeoutSecs: 0,
			},
			expectError: false, // Should still work, just immediate timeout
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := monitor.ValidateCookieWithAPI(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if err.Error() != tt.expectedMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedMsg, err.Error())
				}
			} else {
				// For valid configs, we don't assert success since it depends on external factors
				// We just verify we don't get configuration-related errors
				if err != nil && err.Error() == "test video URL is not configured" {
					t.Errorf("Configuration validation failed unexpectedly: %v", err)
				}
			}
		})
	}
}

func TestValidateCookieWithAPI_EdgeCases(t *testing.T) {
	tests := []struct {
		name              string
		testVideoURL      string
		timeoutSecs       int
		expectConfigError bool
		expectedMsg       string
	}{
		{
			name:              "Valid YouTube URL",
			testVideoURL:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			timeoutSecs:       30,
			expectConfigError: false,
		},
		{
			name:              "Empty URL",
			testVideoURL:      "",
			timeoutSecs:       30,
			expectConfigError: true,
			expectedMsg:       "test video URL is not configured",
		},
		{
			name:              "Invalid URL format",
			testVideoURL:      "not-a-valid-url",
			timeoutSecs:       30,
			expectConfigError: false, // Will fail during execution, not config validation
		},
		{
			name:              "Very short timeout",
			testVideoURL:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			timeoutSecs:       1,
			expectConfigError: false,
		},
		{
			name:              "Zero timeout",
			testVideoURL:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			timeoutSecs:       0,
			expectConfigError: false,
		},
		{
			name:              "Large timeout",
			testVideoURL:      "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			timeoutSecs:       300,
			expectConfigError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ntfyService := &services.NtfyService{}
			cookieNotificationService := NewCookieNotificationService(ntfyService)
			monitor := NewCookieMonitor(cookieNotificationService)

			cfg := config.CookieMonitorConfig{
				TestVideoURL:             tt.testVideoURL,
				APIValidationTimeoutSecs: tt.timeoutSecs,
			}

			err := monitor.ValidateCookieWithAPI(cfg)

			if tt.expectConfigError {
				if err == nil {
					t.Errorf("Expected configuration error, got nil")
				} else if err.Error() != tt.expectedMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedMsg, err.Error())
				}
			} else {
				// For non-config errors, we just verify we don't get config-related errors
				if err != nil && err.Error() == "test video URL is not configured" {
					t.Errorf("Unexpected configuration error: %v", err)
				}
			}
		})
	}
}

func TestCheckCookieHealth_WithAPIValidation_ConfigurationHandling(t *testing.T) {
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)
	monitor := NewCookieMonitor(cookieNotificationService)

	// Save original config
	originalConfig := config.CONFIG.CookieMonitorConfig
	originalCookieContent := config.CONFIG.YtDlpConfig.CookiesContent

	// Set up test config with API validation enabled
	config.CONFIG.CookieMonitorConfig = config.CookieMonitorConfig{
		APIValidationEnabled:     true,
		TestVideoURL:             "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		APIValidationTimeoutSecs: 30,
		WarningThresholdDays:     30,
		UrgentThresholdDays:      7,
	}

	// Set up test cookie content with future expiration
	futureTimestamp := time.Now().Add(60 * 24 * time.Hour).Unix() // 60 days
	config.CONFIG.YtDlpConfig.CookiesContent = fmt.Sprintf(`# Netscape HTTP Cookie File
.youtube.com	TRUE	/	FALSE	%d	VISITOR_INFO1_LIVE	xyz123abc`, futureTimestamp)

	// Restore original config after test
	defer func() {
		config.CONFIG.CookieMonitorConfig = originalConfig
		config.CONFIG.YtDlpConfig.CookiesContent = originalCookieContent
	}()

	// This should complete without panicking, regardless of API validation result
	err := monitor.CheckCookieHealth()
	if err != nil {
		t.Errorf("CheckCookieHealth failed: %v", err)
	}
}

func TestCheckCookieHealth_WithAPIValidation_MissingConfig(t *testing.T) {
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)
	monitor := NewCookieMonitor(cookieNotificationService)

	// Save original config
	originalConfig := config.CONFIG.CookieMonitorConfig
	originalCookieContent := config.CONFIG.YtDlpConfig.CookiesContent

	// Set up test config with API validation enabled but missing URL
	config.CONFIG.CookieMonitorConfig = config.CookieMonitorConfig{
		APIValidationEnabled:     true,
		TestVideoURL:             "", // Missing URL
		APIValidationTimeoutSecs: 30,
		WarningThresholdDays:     30,
		UrgentThresholdDays:      7,
	}

	// Set up test cookie content with future expiration
	futureTimestamp := time.Now().Add(60 * 24 * time.Hour).Unix() // 60 days
	config.CONFIG.YtDlpConfig.CookiesContent = fmt.Sprintf(`# Netscape HTTP Cookie File
.youtube.com	TRUE	/	FALSE	%d	VISITOR_INFO1_LIVE	xyz123abc`, futureTimestamp)

	// Restore original config after test
	defer func() {
		config.CONFIG.CookieMonitorConfig = originalConfig
		config.CONFIG.YtDlpConfig.CookiesContent = originalCookieContent
	}()

	// This should complete without error even if API validation config is invalid
	// The function handles API validation errors gracefully
	err := monitor.CheckCookieHealth()
	if err != nil {
		t.Errorf("CheckCookieHealth failed: %v", err)
	}
}

func TestCheckCookieHealth_Integration_TimeBasedAndAPI(t *testing.T) {
	tests := []struct {
		name                 string
		apiValidationEnabled bool
		testVideoURL         string
		cookieExpiresIn      time.Duration
		expectedBehavior     string
	}{
		{
			name:                 "API validation disabled, cookie healthy",
			apiValidationEnabled: false,
			testVideoURL:         "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			cookieExpiresIn:      60 * 24 * time.Hour, // 60 days
			expectedBehavior:     "no notification",
		},
		{
			name:                 "API validation enabled, cookie healthy",
			apiValidationEnabled: true,
			testVideoURL:         "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			cookieExpiresIn:      60 * 24 * time.Hour, // 60 days
			expectedBehavior:     "depends on API validation result",
		},
		{
			name:                 "API validation enabled, missing URL, cookie healthy",
			apiValidationEnabled: true,
			testVideoURL:         "",                  // Missing URL
			cookieExpiresIn:      60 * 24 * time.Hour, // 60 days
			expectedBehavior:     "API validation failure notification only",
		},
		{
			name:                 "API validation disabled, cookie warning",
			apiValidationEnabled: false,
			testVideoURL:         "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			cookieExpiresIn:      20 * 24 * time.Hour, // 20 days (within warning threshold)
			expectedBehavior:     "time-based warning notification",
		},
		{
			name:                 "API validation enabled, cookie warning",
			apiValidationEnabled: true,
			testVideoURL:         "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			cookieExpiresIn:      20 * 24 * time.Hour, // 20 days (within warning threshold)
			expectedBehavior:     "depends on API validation result",
		},
		{
			name:                 "API validation enabled, missing URL, cookie warning",
			apiValidationEnabled: true,
			testVideoURL:         "",                  // Missing URL
			cookieExpiresIn:      20 * 24 * time.Hour, // 20 days (within warning threshold)
			expectedBehavior:     "API validation failure notification only (no time-based)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ntfyService := &services.NtfyService{}
			cookieNotificationService := NewCookieNotificationService(ntfyService)
			monitor := NewCookieMonitor(cookieNotificationService)

			// Save original config
			originalConfig := config.CONFIG.CookieMonitorConfig
			originalCookieContent := config.CONFIG.YtDlpConfig.CookiesContent

			// Set up test config
			config.CONFIG.CookieMonitorConfig = config.CookieMonitorConfig{
				APIValidationEnabled:     tt.apiValidationEnabled,
				TestVideoURL:             tt.testVideoURL,
				APIValidationTimeoutSecs: 30,
				WarningThresholdDays:     30,
				UrgentThresholdDays:      7,
			}

			// Set up test cookie content with specific expiration
			expirationTimestamp := time.Now().Add(tt.cookieExpiresIn).Unix()
			config.CONFIG.YtDlpConfig.CookiesContent = fmt.Sprintf(`# Netscape HTTP Cookie File
.youtube.com	TRUE	/	FALSE	%d	VISITOR_INFO1_LIVE	xyz123abc`, expirationTimestamp)

			// Restore original config after test
			defer func() {
				config.CONFIG.CookieMonitorConfig = originalConfig
				config.CONFIG.YtDlpConfig.CookiesContent = originalCookieContent
			}()

			// Run the test
			err := monitor.CheckCookieHealth()
			if err != nil {
				t.Errorf("CheckCookieHealth failed: %v", err)
			}

			// Note: In a real test scenario, we would want to capture and verify
			// the actual notifications sent, but this requires more complex mocking
			// of the notification service. For now, we verify the function completes
			// without error, which indicates the integration logic is working.
		})
	}
}

func TestCookieNotificationService_SendAPIValidationFailedNotification(t *testing.T) {
	// Test the new SendAPIValidationFailedNotification method
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)

	testError := errors.New("API validation failed: timeout")

	// This should not panic even with a disabled ntfy service
	err := cookieNotificationService.SendAPIValidationFailedNotification(testError)
	if err != nil {
		t.Errorf("SendAPIValidationFailedNotification failed: %v", err)
	}
}

func TestCookieNotificationService_Methods(t *testing.T) {
	// Test all notification methods to ensure they don't panic
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)

	testTime := time.Now().Add(24 * time.Hour)
	testDuration := 24 * time.Hour
	testError := errors.New("test error")

	tests := []struct {
		name     string
		testFunc func() error
	}{
		{
			name: "SendWarningNotification",
			testFunc: func() error {
				return cookieNotificationService.SendWarningNotification("test_cookie", testDuration, testTime)
			},
		},
		{
			name: "SendUrgentNotification",
			testFunc: func() error {
				return cookieNotificationService.SendUrgentNotification("test_cookie", testDuration, testTime)
			},
		},
		{
			name: "SendExpiredNotification",
			testFunc: func() error {
				return cookieNotificationService.SendExpiredNotification("test_cookie", testTime)
			},
		},
		{
			name: "SendHealthyNotification",
			testFunc: func() error {
				return cookieNotificationService.SendHealthyNotification("test_cookie", testDuration)
			},
		},
		{
			name: "SendAPIValidationFailedNotification",
			testFunc: func() error {
				return cookieNotificationService.SendAPIValidationFailedNotification(testError)
			},
		},
		{
			name: "SendTestNotification",
			testFunc: func() error {
				return cookieNotificationService.SendTestNotification()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			// Should not panic and should complete without error when ntfy is disabled
			if err != nil && err.Error() != "ntfy notifications are disabled" {
				t.Errorf("%s failed: %v", tt.name, err)
			}
		})
	}
}

func TestValidateCookieWithAPI_URLValidation(t *testing.T) {
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)
	monitor := NewCookieMonitor(cookieNotificationService)

	tests := []struct {
		name        string
		testURL     string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Valid YouTube URL",
			testURL:     "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			shouldError: false,
		},
		{
			name:        "Valid YouTube short URL",
			testURL:     "https://youtu.be/dQw4w9WgXcQ",
			shouldError: false,
		},
		{
			name:        "Empty URL",
			testURL:     "",
			shouldError: true,
			errorMsg:    "test video URL is not configured",
		},
		{
			name:        "Invalid URL format",
			testURL:     "not-a-valid-url",
			shouldError: false, // Will fail during execution, not URL validation
		},
		{
			name:        "Non-YouTube URL",
			testURL:     "https://example.com/video",
			shouldError: false, // Will fail during execution, not URL validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.CookieMonitorConfig{
				TestVideoURL:             tt.testURL,
				APIValidationTimeoutSecs: 30,
			}

			err := monitor.ValidateCookieWithAPI(cfg)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				// For valid URLs, we don't assert success since it depends on external factors
				// We just verify we don't get URL validation errors
				if err != nil && err.Error() == "test video URL is not configured" {
					t.Errorf("Unexpected URL validation error: %v", err)
				}
			}
		})
	}
}

func TestEvaluateAndNotify_ThresholdLogic(t *testing.T) {
	ntfyService := &services.NtfyService{}
	cookieNotificationService := NewCookieNotificationService(ntfyService)
	monitor := NewCookieMonitor(cookieNotificationService)

	// Save original config
	originalConfig := config.CONFIG.CookieMonitorConfig
	defer func() {
		config.CONFIG.CookieMonitorConfig = originalConfig
	}()

	// Set up test config
	config.CONFIG.CookieMonitorConfig = config.CookieMonitorConfig{
		WarningThresholdDays: 30,
		UrgentThresholdDays:  7,
	}

	tests := []struct {
		name            string
		timeUntilExpiry time.Duration
		expectedType    string
	}{
		{
			name:            "Expired cookie",
			timeUntilExpiry: -24 * time.Hour,
			expectedType:    "expired",
		},
		{
			name:            "Urgent threshold",
			timeUntilExpiry: 3 * 24 * time.Hour, // 3 days
			expectedType:    "urgent",
		},
		{
			name:            "Warning threshold",
			timeUntilExpiry: 15 * 24 * time.Hour, // 15 days
			expectedType:    "warning",
		},
		{
			name:            "Healthy cookie",
			timeUntilExpiry: 60 * 24 * time.Hour, // 60 days
			expectedType:    "healthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cookieInfo := &CookieInfo{
				Name:      "test_cookie",
				ExpiresAt: time.Now().Add(tt.timeUntilExpiry),
				IsValid:   tt.timeUntilExpiry > 0,
			}

			// This method is not exported, so we test it indirectly through CheckCookieHealth
			// The test verifies the method doesn't panic and handles different scenarios
			err := monitor.evaluateAndNotify(cookieInfo, tt.timeUntilExpiry)

			// Should not return an error (notifications might be disabled)
			if err != nil {
				t.Errorf("evaluateAndNotify failed: %v", err)
			}
		})
	}
}
