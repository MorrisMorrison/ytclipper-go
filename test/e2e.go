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

	validYouTubeURL         = "https://www.youtube.com/watch?v=hf_HZZgdrJ8"
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

	err := chromedp.Run(ctx,
		// Step 1: Navigate to the app
		chromedp.Navigate(baseURL),

		// Step 2: Fill the YouTube URL
		chromedp.WaitVisible(urlInputSelector, chromedp.ByID),
		chromedp.SetValue(urlInputSelector, validYouTubeURL, chromedp.ByID),

		// Step 4: Wait for the dropdown to be enabled
		chromedp.WaitEnabled(formatSelectSelector, chromedp.ByID),

		// Step 5: Select a format (e.g., the first available option)
		chromedp.SetValue(formatSelectSelector, validFormatValue, chromedp.ByID),

		// Step 6: Fill the "From" and "To" inputs
		chromedp.SendKeys(fromInputSelector, validFromTimestamp, chromedp.ByID),
		chromedp.SendKeys(toInputSelector, validToTimestamp, chromedp.ByID),

		// Step 7: Click the "Clip!" button
		chromedp.Click(clipButtonSelector, chromedp.ByID),

		// Step 8: Wait for the download link to appear
		chromedp.WaitVisible(downloadLinkWrapperSel, chromedp.ByID),

		// Step 9: Extract the download link text
		chromedp.AttributeValue(downloadLinkSelector, "href", &downloadLink, nil),
	)
	if err != nil {
		return fmt.Errorf("workflow error: %w", err)
	}

	// Validate the result
	if downloadLink == "" {
		return fmt.Errorf("download link was not generated")
	}

	log.Printf("Download link: %s", downloadLink)
	return nil
}

func testInvalidYouTubeURL(ctx context.Context) error {
	err := chromedp.Run(ctx,
		// Step 1: Navigate to the app
		chromedp.Navigate(baseURL),

		// Step 2: Enter an invalid YouTube URL
		chromedp.WaitVisible(urlInputSelector, chromedp.ByID),
		chromedp.SetValue(urlInputSelector, invalidYouTubeURL, chromedp.ByID),

		// Step 4: Check for an error message
		chromedp.WaitVisible(errorMessageSelector, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("workflow error: %w", err)
	}

	log.Printf("Invalid YouTube URL test passed")
	return nil
}

func testInvalidTimestamps(ctx context.Context) error {
    err := chromedp.Run(ctx,
        // Step 1: Navigate to the app
        chromedp.Navigate(baseURL),

        // Step 2: Enter a valid YouTube URL
		chromedp.SetValue(urlInputSelector, validYouTubeURL, chromedp.ByID),

        chromedp.WaitEnabled(formatSelectSelector, chromedp.ByID),
        chromedp.SetValue(formatSelectSelector, validFormatValue, chromedp.ByID),

        // Step 3: Enter invalid "From" and "To" values
        chromedp.SendKeys(fromInputSelector, fromInvalidTimestamp, chromedp.ByID),
        chromedp.SendKeys(toInputSelector, toInvalidTimestamp, chromedp.ByID),

        // Step 4: Attempt to clip
        chromedp.Click(clipButtonSelector, chromedp.ByID),

        // Step 5: Check for error message
        chromedp.WaitVisible(errorMessageSelector, chromedp.ByQuery),
    )
    if err != nil {
        return fmt.Errorf("workflow error: %w", err)
    }

    log.Printf("Invalid timestamps test passed")
    return nil
}


func testDarkModeToggle(ctx context.Context) error {
    var initialClass, toggledClass string
    var initialBackgroundColor, toggledBackgroundColor string

    err := chromedp.Run(ctx,
        // Step 1: Navigate to the app
        chromedp.Navigate(baseURL),

        // Step 2: Get the initial class of the <body> element
        chromedp.AttributeValue(`body`, "class", &initialClass, nil),

        // Step 3: Get the initial background color
        chromedp.Evaluate(`window.getComputedStyle(document.body).backgroundColor`, &initialBackgroundColor),

        // Step 4: Toggle the dark mode slider
        chromedp.Click(`#themeSlider`, chromedp.ByID),

        // Step 5: Get the toggled class of the <body> element
        chromedp.AttributeValue(`body`, "class", &toggledClass, nil),

        // Step 6: Get the toggled background color
        chromedp.Evaluate(`window.getComputedStyle(document.body).backgroundColor`, &toggledBackgroundColor),
    )
    if err != nil {
        return fmt.Errorf("workflow error: %w", err)
    }

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
