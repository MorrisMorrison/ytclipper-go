package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	baseURL                = "http://localhost:8080"
	urlInputSelector       = `#url`
	previewButtonSelector  = `#previewButton`
	formatSelectSelector   = `#formatSelect`
	fromInputSelector      = `#from`
	toInputSelector        = `#to`
	clipButtonSelector     = `#clipButton`
	downloadLinkWrapperSel = `#downloadLinkWrapper`
	downloadLinkSelector   = `#downloadLink`
	errorMessageSelector   = `.toast-error`

	invalidURLMessage       = "Invalid Url"
	invalidInputMessage     = "Invalid input"
	validYouTubeURL         = "https://www.youtube.com/watch?v=hf_HZZgdrJ8"
	invalidYouTubeURL       = "invalid-url"
	fromInvalidTimestamp    = "1231:111"
	toInvalidTimestamp      = "111"
	validFromTimestamp      = "20"
	validToTimestamp        = "40"
	validFormatValue        = "136"
)

type Test struct {
	Name string
	Run  func(ctx context.Context) error
}

func main() {
	tests := []Test{
		{
			Name: "Invalid YouTube URL Test",
			Run:  testInvalidYouTubeURL,
		},
		// {
		// 	Name: "Invalid Timestamps Test",
		// 	Run:  testInvalidTimestamps,
		// },
		{
			Name: "Dark Mode Test",
			Run:  testDarkModeToggle,
		},
		{
			Name: "Basic Workflow Test",
			Run:  testBasicWorkflow,
		},
	}

	var failedTests int
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	for _, test := range tests {

		log.Printf("Running test: %s", test.Name)
		err := test.Run(ctx)
		if err != nil {
			log.Printf("Test '%s' failed: %v", test.Name, err)
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
	os.Exit(0)
}

func testBasicWorkflow(ctx context.Context) error {
	var downloadLink string

	err := chromedp.Run(ctx,
		// Step 1: Navigate to the app
		chromedp.Navigate(baseURL),

		// Step 2: Fill the YouTube URL
		chromedp.WaitVisible(urlInputSelector, chromedp.ByID),
		chromedp.SendKeys(urlInputSelector, validYouTubeURL, chromedp.ByID),

		// Step 3: Click the preview button to fetch formats
		chromedp.Click(previewButtonSelector, chromedp.ByID),

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
	var errorMessage string

	err := chromedp.Run(ctx,
		// Step 1: Navigate to the app
		chromedp.Navigate(baseURL),

		// Step 2: Enter an invalid YouTube URL
		chromedp.WaitVisible(urlInputSelector, chromedp.ByID),
		chromedp.SendKeys(urlInputSelector, invalidYouTubeURL, chromedp.ByID),

		// Step 3: Click the preview button
		chromedp.Click(previewButtonSelector, chromedp.ByID),

		// Step 4: Check for an error message
		chromedp.WaitVisible(errorMessageSelector, chromedp.ByQuery),
		chromedp.Text(errorMessageSelector, &errorMessage, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("workflow error: %w", err)
	}

	if !strings.Contains(errorMessage, invalidURLMessage) {
		return fmt.Errorf("unexpected error message: %s", errorMessage)
	}

	log.Printf("Invalid YouTube URL test passed")
	return nil
}

func testInvalidTimestamps(ctx context.Context) error {
    var errorMessage string

    testCtx, testCancel := context.WithTimeout(ctx, 10*time.Second)
    defer testCancel()

    err := chromedp.Run(testCtx,
        // Step 1: Navigate to the app
        chromedp.Navigate(baseURL),

        // Step 2: Enter a valid YouTube URL
        chromedp.SendKeys(urlInputSelector, validYouTubeURL, chromedp.ByID),
        chromedp.Click(previewButtonSelector, chromedp.ByID),
        chromedp.WaitEnabled(formatSelectSelector, chromedp.ByID),
        chromedp.SetValue(formatSelectSelector, validFormatValue, chromedp.ByID),

        // Step 3: Enter invalid "From" and "To" values
        chromedp.SendKeys(fromInputSelector, fromInvalidTimestamp, chromedp.ByID),
        chromedp.SendKeys(toInputSelector, toInvalidTimestamp, chromedp.ByID),

        // Step 4: Attempt to clip
        chromedp.Click(clipButtonSelector, chromedp.ByID),

        // Step 5: Check for error message
        chromedp.WaitVisible(errorMessageSelector, chromedp.ByQuery),
        chromedp.Text(errorMessageSelector, &errorMessage, chromedp.ByQuery),
    )
    if err != nil {
        if testCtx.Err() == context.DeadlineExceeded {
            return fmt.Errorf("test timed out")
        }
        return fmt.Errorf("workflow error: %w", err)
    }

    if !strings.Contains(errorMessage, invalidInputMessage) {
        return fmt.Errorf("unexpected error message: %s", errorMessage)
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
