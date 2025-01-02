package main

import (
	"context"
	"fmt"
	"log"
	"os"
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

	validYouTubeURL         = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	invalidYouTubeURL       = "invalid-url"
	fromInvalidTimestamp    = "1231:111"
	toInvalidTimestamp      = "111"
	validFromTimestamp      = "10"
	validToTimestamp        = "20"
	validFormatValue        = "136"
)

type Test struct {
	Name string
	Run  func(ctx context.Context) error
}

func main() {
	tests := []Test{
		{Name: "Invalid Timestamps Test", Run: testInvalidTimestamps},
		{Name: "Invalid YouTube URL Test", Run: testInvalidYouTubeURL},
		{Name: "Dark Mode Test", Run: testDarkModeToggle},
		{Name: "Basic Workflow Test", Run: testBasicWorkflow},
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Headless, // Disable for debugging
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("v", true), // Verbose logging
	)

	log.Println("Starting Chrome context with options:", opts)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	var failedTests int
	for _, test := range tests {
		ctx, cancel := chromedp.NewContext(allocCtx)
		defer cancel()

		testCtx, cancelTest := context.WithTimeout(ctx, 2*time.Minute)
		defer cancelTest()

		log.Printf("Running test: %s", test.Name)
		err := test.Run(testCtx)
		if err != nil {
			log.Printf("Test '%s' failed: %v\nDetails: %v", test.Name, err, testCtx.Err())
			failedTests++
		} else {
			log.Printf("Test '%s' passed", test.Name)
		}
	}

	if failedTests > 0 {
		log.Printf("%d test(s) failed", failedTests)
		os.Exit(1)
	}
	log.Println("All tests passed")
}


func testBasicWorkflow(ctx context.Context) error {
	var downloadLink string

	log.Printf("Navigating to %s", baseURL)
	err := chromedp.Run(ctx,
		// Step 1: Navigate to the app
		chromedp.Navigate(baseURL),
	)
	if err != nil {
		return fmt.Errorf("failed to navigate to app: %w", err)
	}
	log.Println("Successfully navigated to the app")

	log.Printf("Filling YouTube URL: %s", validYouTubeURL)
	err = chromedp.Run(ctx,
		// Step 2: Fill the YouTube URL
		chromedp.WaitVisible(urlInputSelector, chromedp.ByID),
		chromedp.SetValue(urlInputSelector, validYouTubeURL, chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("failed to fill YouTube URL: %w", err)
	}
	log.Println("YouTube URL entered")

	log.Println("Waiting for format dropdown to be enabled")
	err = chromedp.Run(ctx,
		// Step 4: Wait for the dropdown to be enabled
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

	log.Println("Waiting for download link to appear")
	err = chromedp.Run(ctx,
		// Step 8: Wait for the download link to appear
		chromedp.WaitVisible(downloadLinkWrapperSel, chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("download link did not appear: %w", err)
	}
	log.Println("Download link wrapper visible")

	log.Println("Extracting download link")
	err = chromedp.Run(ctx,
		// Step 9: Extract the download link text
		chromedp.AttributeValue(downloadLinkSelector, "href", &downloadLink, nil),
	)
	if err != nil {
		return fmt.Errorf("failed to extract download link: %w", err)
	}

	// Validate the result
	if downloadLink == "" {
		return fmt.Errorf("download link was not generated")
	}

	log.Printf("Download link successfully generated: %s", downloadLink)
	return nil
}


func testInvalidYouTubeURL(ctx context.Context) error {
	log.Printf("Navigate to %s", baseURL)

	err := chromedp.Run(ctx,
		// Step 1: Navigate to the app
		chromedp.Navigate(baseURL),
	)
	if err != nil {
		return fmt.Errorf("failed to navigate to app: %w", err)
	}
	log.Println("Successfully navigated to the app")

	log.Printf("Entering invalid YouTube URL: %s", invalidYouTubeURL)
	err = chromedp.Run(ctx,
		// Step 2: Enter an invalid YouTube URL
		chromedp.WaitVisible(urlInputSelector, chromedp.ByID),
		chromedp.SetValue(urlInputSelector, invalidYouTubeURL, chromedp.ByID),
	)
	if err != nil {
		return fmt.Errorf("failed to enter invalid YouTube URL: %w", err)
	}
	log.Println("Invalid YouTube URL entered")

	log.Println("Waiting for error message to appear")
	err = chromedp.Run(ctx,
		// Step 4: Check for an error message
		chromedp.WaitVisible(errorMessageSelector, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("error message did not appear: %w", err)
	}
	log.Println("Error message displayed as expected")

	log.Printf("Invalid YouTube URL test passed")
	return nil
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
        chromedp.SetValue(urlInputSelector, validYouTubeURL, chromedp.ByID),
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


func testDarkModeToggle(ctx context.Context) error {
    var initialClass, toggledClass string
    var initialBackgroundColor, toggledBackgroundColor string

    log.Printf("Navigating to %s", baseURL)
    err := chromedp.Run(ctx, chromedp.Navigate(baseURL))
    if err != nil {
        return fmt.Errorf("failed to navigate to app: %w", err)
    }
    log.Println("Successfully navigated to the app")

    log.Println("Fetching initial class of the <body> element")
    err = chromedp.Run(ctx, chromedp.AttributeValue(`body`, "class", &initialClass, nil))
    if err != nil {
        return fmt.Errorf("failed to fetch initial body class: %w", err)
    }
    log.Printf("Initial body class: %s", initialClass)

    log.Println("Fetching initial background color of the <body>")
    err = chromedp.Run(ctx, chromedp.Evaluate(`window.getComputedStyle(document.body).backgroundColor`, &initialBackgroundColor))
    if err != nil {
        return fmt.Errorf("failed to fetch initial background color: %w", err)
    }
    log.Printf("Initial background color: %s", initialBackgroundColor)

    log.Println("Toggling the dark mode slider")
    err = chromedp.Run(ctx, chromedp.Click(`#themeSlider`, chromedp.ByID), chromedp.Sleep(500*time.Millisecond))
    if err != nil {
        return fmt.Errorf("failed to toggle the dark mode slider: %w", err)
    }
    log.Println("Dark mode slider toggled")

    log.Println("Fetching toggled class of the <body> element")
    err = chromedp.Run(ctx, chromedp.AttributeValue(`body`, "class", &toggledClass, nil))
    if err != nil {
        return fmt.Errorf("failed to fetch toggled body class: %w", err)
    }
    log.Printf("Toggled body class: %s", toggledClass)

    log.Println("Fetching toggled background color of the <body>")
    err = chromedp.Run(ctx, chromedp.Evaluate(`window.getComputedStyle(document.body).backgroundColor`, &toggledBackgroundColor))
    if err != nil {
        return fmt.Errorf("failed to fetch toggled background color: %w", err)
    }
    log.Printf("Toggled background color: %s", toggledBackgroundColor)

    if initialClass == toggledClass {
        return fmt.Errorf("dark mode toggle did not update the body class: initial=%s, toggled=%s", initialClass, toggledClass)
    }

    if initialBackgroundColor == toggledBackgroundColor {
        return fmt.Errorf("dark mode toggle did not change the background color: initial=%s, toggled=%s", initialBackgroundColor, toggledBackgroundColor)
    }

    log.Printf("Dark mode toggle test passed: class changed from '%s' to '%s', background color changed from '%s' to '%s'",
        initialClass, toggledClass, initialBackgroundColor, toggledBackgroundColor)
    return nil
}
