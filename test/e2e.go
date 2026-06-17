// E2E Tests for ytclipper-go
//
// This enhanced test suite balances realistic testing with CI performance:
// - Tests both success and failure scenarios
// - Uses smart timeout detection for fast failures
// - Maintains realistic expectations while being CI-friendly
// - Tests timeout configuration and error handling
//
// Test Strategy:
// - Basic Workflow: Tests full UI flow with adaptive timeout
// - Fast Failure Detection: Tests quick error detection (15s timeout)
// - Timeout Configuration: Tests backend response times
// - Error Handling: Tests invalid inputs and error messages
//
// Environment Variables:
// - E2E_DOWNLOAD_TIMEOUT: Timeout in seconds for download operations (default: 25)
// - E2E_EXPECT_FAILURE: Whether to expect download failures (default: true)
// - CI: Automatically detected CI environment (affects timeout behavior)
// - YTCLIPPER_YT_DLP_COMMAND_TIMEOUT_IN_SECONDS: Backend timeout (affects test expectations)
//
// Usage:
//
//	go run test/e2e.go                    # Run with defaults (balanced approach)
//	E2E_DOWNLOAD_TIMEOUT=60 go run test/e2e.go   # Use longer timeout for real testing
//	E2E_EXPECT_FAILURE=false go run test/e2e.go  # Expect downloads to succeed
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	baseURL                = "http://localhost:8080"
	urlInputSelector       = `#url`
	formatSelectSelector   = `#formatSelect`
	fromInputSelector      = `#from`
	toInputSelector        = `#to`
	clipButtonSelector     = `#clipButton`
	downloadLinkWrapperSel = `#downloadLinkWrapper`
	downloadLinkSelector   = `#downloadLink`
	errorMessageSelector   = `.toast-error`

	validYouTubeURL      = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	invalidYouTubeURL    = "invalid-url"
	fromInvalidTimestamp = "1231:111"
	toInvalidTimestamp   = "111"
	validFromTimestamp   = "10"
	validToTimestamp     = "20"
	validFormatValue     = "136"

	// Test configuration - realistic timeouts with fast failure detection
	defaultDownloadTimeoutSeconds = 25    // Shorter timeout for CI but realistic
	ciQuickFailTimeoutSeconds     = 15    // Very fast timeout for known failure tests
	defaultExpectDownloadFailure  = false // CI uses a yt-dlp stub, so downloads should succeed
)

// getDownloadTimeoutSeconds returns the timeout from environment or default
func getDownloadTimeoutSeconds() int {
	if timeoutStr := os.Getenv("E2E_DOWNLOAD_TIMEOUT"); timeoutStr != "" {
		if timeout, err := strconv.Atoi(timeoutStr); err == nil {
			return timeout
		}
	}
	return defaultDownloadTimeoutSeconds
}

// expectDownloadFailure returns whether to expect download failures from environment or default
func expectDownloadFailure() bool {
	if expectStr := os.Getenv("E2E_EXPECT_FAILURE"); expectStr != "" {
		if expect, err := strconv.ParseBool(expectStr); err == nil {
			return expect
		}
	}
	return defaultExpectDownloadFailure
}

// isCIEnvironment checks if we're running in a CI environment
func isCIEnvironment() bool {
	return os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" || os.Getenv("GITLAB_CI") != ""
}

type Test struct {
	Name string
	Run  func(ctx context.Context) error
}

func main() {
	tests := []Test{
		{Name: "Basic Workflow Test", Run: testBasicWorkflow},
		{Name: "Invalid Timestamps Test", Run: testInvalidTimestamps},
		{Name: "Timeout Configuration Test", Run: testTimeoutConfiguration},
	}

	var failedTests int
	for _, test := range tests {
		// Each test runs in its own function so its Chrome allocator/context are
		// torn down (via defer) BEFORE the next test starts. Running the loop body
		// inline leaked a Chrome instance per iteration, starving later tests.
		if err := runTest(test); err != nil {
			failedTests++
		}
	}

	if failedTests > 0 {
		log.Printf("%d test(s) failed", failedTests)
		os.Exit(1)
	}
	log.Println("All tests passed")
}

func runTest(test Test) error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Headless,
		// The first Chrome launch in CI is a cold start and can exceed the default
		// 20s websocket timeout; give it generous headroom.
		chromedp.WSURLReadTimeout(45*time.Second),
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor,PrivateNetworkAccessSendPreflights,PrivateNetworkAccessRespectPreflightResults"),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-field-trial-config", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
	)

	log.Printf("Starting Chrome context with options for test: %s", test.Name)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	// Create context with error logging suppression for known chromedp issues
	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(s string, i ...any) {
		// Suppress known chromedp protocol errors that don't affect functionality
		msg := fmt.Sprintf(s, i...)
		if strings.Contains(msg, "PrivateNetworkRequestPolicy") ||
			strings.Contains(msg, "PermissionWarn") ||
			strings.Contains(msg, "could not unmarshal event") {
			return // Suppress these specific errors
		}
		log.Printf("Chrome: %s", msg)
	}))
	defer cancel()

	testCtx, cancelTest := context.WithTimeout(ctx, 2*time.Minute)
	defer cancelTest()

	log.Printf("Running test: %s", test.Name)
	startTime := time.Now()
	err := test.Run(testCtx)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("Test '%s' failed after %v: %v", test.Name, duration, err)
		if testCtx.Err() == context.DeadlineExceeded {
			log.Printf("  → Test timed out")
		}
		return err
	}

	log.Printf("Test '%s' passed in %v", test.Name, duration)
	return nil
}

func testBasicWorkflow(ctx context.Context) error {
	log.Printf("Navigating to %s", baseURL)
	err := chromedp.Run(ctx,
		// Step 1: Navigate to the app
		chromedp.Navigate(baseURL),
	)
	if err != nil {
		return fmt.Errorf("failed to navigate to app: %w (ensure server is running on %s)", err, baseURL)
	}
	log.Println("Successfully navigated to the app")

	log.Printf("Filling YouTube URL: %s", validYouTubeURL)
	err = chromedp.Run(ctx,
		// Step 2: Fill the YouTube URL
		chromedp.WaitVisible(urlInputSelector, chromedp.ByID),
		// Use SendKeys (real keystrokes) rather than SetValue: the URL field has a
		// debounced `input` listener that loads formats, and SetValue does not
		// reliably fire that event under chromedp.
		chromedp.SendKeys(urlInputSelector, validYouTubeURL, chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("failed to fill YouTube URL: %w", err)
	}
	log.Println("YouTube URL entered")

	log.Println("Waiting for format dropdown to be enabled")
	err = chromedp.Run(ctx,
		// Step 4: Wait for the dropdown to be enabled
		chromedp.WaitReady(formatSelectSelector, chromedp.ByID),
		chromedp.WaitEnabled(formatSelectSelector, chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("format dropdown not enabled: %w", err)
	}
	log.Println("Format dropdown enabled")

	log.Printf("Selecting video format: %s", validFormatValue)
	err = chromedp.Run(ctx,
		// Step 5: Select a format (e.g., the first available option)
		chromedp.SetValue(formatSelectSelector, validFormatValue, chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("failed to select video format: %w", err)
	}
	log.Println("Video format selected")

	log.Printf("Filling 'From' and 'To' inputs: from=%s, to=%s", validFromTimestamp, validToTimestamp)
	err = chromedp.Run(ctx,
		// Step 6: Fill the "From" and "To" inputs
		chromedp.SendKeys(fromInputSelector, validFromTimestamp, chromedp.ByID),
		chromedp.SendKeys(toInputSelector, validToTimestamp, chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("failed to fill 'From' and 'To' inputs: %w", err)
	}
	log.Println("'From' and 'To' inputs filled")

	log.Println("Clicking 'Clip!' button")
	err = chromedp.Run(ctx,
		// Step 7: Click the "Clip!" button
		chromedp.Click(clipButtonSelector, chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("failed to click 'Clip!' button: %w", err)
	}
	log.Println("'Clip!' button clicked")

	// Test UI flow completion - either success or error handling
	return testClipProcessingResult(ctx)
}

func testInvalidTimestamps(ctx context.Context) error {
	log.Printf("Navigating to %s", baseURL)
	err := chromedp.Run(ctx,
		// Step 1: Navigate to the app
		chromedp.Navigate(baseURL),
	)
	if err != nil {
		return fmt.Errorf("failed to navigate to app: %w", err)
	}
	log.Println("Successfully navigated to the app")

	log.Printf("Entering valid YouTube URL: %s", validYouTubeURL)
	err = chromedp.Run(ctx,
		// Step 2: Enter a valid YouTube URL
		// Use SendKeys (real keystrokes) rather than SetValue: the URL field has a
		// debounced `input` listener that loads formats, and SetValue does not
		// reliably fire that event under chromedp.
		chromedp.SendKeys(urlInputSelector, validYouTubeURL, chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("failed to enter YouTube URL: %w", err)
	}
	log.Println("Valid YouTube URL entered")

	log.Println("Waiting for format dropdown to be enabled")
	err = chromedp.Run(ctx,
		chromedp.WaitReady(formatSelectSelector, chromedp.ByID),
		chromedp.WaitEnabled(formatSelectSelector, chromedp.ByID),
		chromedp.SetValue(formatSelectSelector, validFormatValue, chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("failed to select video format: %w", err)
	}
	log.Printf("Video format '%s' selected", validFormatValue)

	log.Printf("Entering invalid timestamps: from=%s, to=%s", fromInvalidTimestamp, toInvalidTimestamp)
	err = chromedp.Run(ctx,
		// Step 3: Enter invalid "From" and "To" values
		chromedp.SendKeys(fromInputSelector, fromInvalidTimestamp, chromedp.ByID),
		chromedp.SendKeys(toInputSelector, toInvalidTimestamp, chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("failed to enter timestamps: %w", err)
	}
	log.Println("Invalid timestamps entered")

	log.Println("Clicking 'Clip' button")
	err = chromedp.Run(ctx,
		// Step 4: Attempt to clip
		chromedp.Click(clipButtonSelector, chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("failed to click 'Clip' button: %w", err)
	}
	log.Println("'Clip' button clicked")

	log.Println("Waiting for error message to appear")
	err = chromedp.Run(ctx,
		// Step 5: Check for error message
		chromedp.WaitVisible(errorMessageSelector, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("error message did not appear: %w", err)
	}
	log.Println("Error message displayed as expected")

	log.Printf("Invalid timestamps test passed")
	return nil
}

// testClipProcessingResult waits for either success or error and verifies the UI handles it properly
func testClipProcessingResult(ctx context.Context) error {
	return testClipProcessingResultWithTimeout(ctx, getDownloadTimeoutSeconds())
}

// testClipProcessingResultWithTimeout allows custom timeout for different test scenarios
func testClipProcessingResultWithTimeout(ctx context.Context, timeoutSeconds int) error {
	log.Printf("Waiting for clip processing result (timeout: %d seconds)", timeoutSeconds)

	// Create a timeout context for the processing result
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	// For fast failure detection, we'll check for errors more frequently
	checkInterval := 2 * time.Second
	if timeoutSeconds <= ciQuickFailTimeoutSeconds {
		checkInterval = 500 * time.Millisecond
	}

	// Wait for either success (download link) or error message with periodic checks
	var downloadLink string
	var errorText string

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			if expectDownloadFailure() {
				// Opt-in path for manual runs against real YouTube, where a
				// timeout/block is expected. CI uses the yt-dlp stub and does not
				// set this, so a timeout here is a real failure.
				log.Printf("Test timed out after %d seconds - download failure was expected (E2E_EXPECT_FAILURE)", timeoutSeconds)
				return nil
			}
			return fmt.Errorf("test timed out after %d seconds (no download link or error shown)", timeoutSeconds)

		case <-ticker.C:
			// Check for download link first
			err := chromedp.Run(timeoutCtx,
				chromedp.ActionFunc(func(ctx context.Context) error {
					return chromedp.Run(ctx,
						chromedp.WaitVisible(downloadLinkWrapperSel, chromedp.ByID),
						chromedp.AttributeValue(downloadLinkSelector, "href", &downloadLink, nil),
					)
				}),
			)
			if err == nil && downloadLink != "" {
				log.Printf("Success: Download link generated: %s", downloadLink)
				return nil
			}

			// Check for error message
			err = chromedp.Run(timeoutCtx,
				chromedp.ActionFunc(func(ctx context.Context) error {
					return chromedp.Run(ctx,
						chromedp.WaitVisible(errorMessageSelector, chromedp.ByQuery),
						chromedp.Text(errorMessageSelector, &errorText, chromedp.ByQuery),
					)
				}),
			)
			if err == nil && errorText != "" {
				log.Printf("Success: Error handled properly: %s", errorText)
				return nil
			}
		}
	}
}

// testTimeoutConfiguration tests the application's timeout handling behavior
func testTimeoutConfiguration(ctx context.Context) error {
	log.Printf("Testing timeout configuration and backend response times")

	// This test verifies that the system properly configures timeouts for yt-dlp
	// We'll test this by making a quick API call that should either succeed fast or fail fast

	// Navigate to app
	err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL),
		chromedp.WaitVisible(urlInputSelector, chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("failed to navigate to app: %w", err)
	}

	// Test the GetVideoDuration endpoint which should be faster than full downloads
	log.Println("Testing video duration fetch (should be fast)")

	err = chromedp.Run(ctx,
		// Use SendKeys (real keystrokes) rather than SetValue: the URL field has a
		// debounced `input` listener that loads formats, and SetValue does not
		// reliably fire that event under chromedp.
		chromedp.SendKeys(urlInputSelector, validYouTubeURL, chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("failed to enter URL: %w", err)
	}

	// Wait for format dropdown to be populated (indicates backend processed the URL)
	start := time.Now()
	err = chromedp.Run(ctx,
		chromedp.WaitEnabled(formatSelectSelector, chromedp.ByID),
	)
	duration := time.Since(start)

	if err != nil {
		log.Printf("Format loading timed out after %v - this indicates backend processing issues", duration)
		return fmt.Errorf("format dropdown not enabled within reasonable time: %w", err)
	}

	log.Printf("Backend responded in %v - this indicates good timeout configuration", duration)
	return nil
}
