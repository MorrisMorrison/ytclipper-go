package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/chromedp/chromedp"
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
		{
			Name: "Invalid Timestamps Test",
			Run:  testInvalidTimestamps,
		},
		{
			Name: "Basic Workflow Test",
			Run:  testBasicWorkflow,
		},
	}

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var failedTests int

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

// testBasicWorkflow simulates the workflow of filling inputs and downloading a clip
func testBasicWorkflow(ctx context.Context) error {
	var downloadLink string

	err := chromedp.Run(ctx,
		// Step 1: Navigate to the app
		chromedp.Navigate("http://localhost:8080"),

		// Step 2: Fill the YouTube URL
		chromedp.WaitVisible(`#url`, chromedp.ByID),
		chromedp.SendKeys(`#url`, "https://www.youtube.com/watch?v=hf_HZZgdrJ8", chromedp.ByID),

		// Step 3: Click the preview button to fetch formats
		chromedp.Click(`#previewButton`, chromedp.ByID),

		// Step 4: Wait for the dropdown to be enabled
		chromedp.WaitEnabled(`#formatSelect`, chromedp.ByID),

		// Step 5: Select a format (e.g., the first available option)
		chromedp.SetValue(`#formatSelect`, "136", chromedp.ByID),

		// Step 6: Fill the "From" and "To" inputs
		chromedp.SendKeys(`#from`, "20", chromedp.ByID),
		chromedp.SendKeys(`#to`, "40", chromedp.ByID),

		// Step 7: Click the "Clip!" button
		chromedp.Click(`#clipButton`, chromedp.ByID),

		// Step 8: Wait for the download link to appear
		chromedp.WaitVisible(`#downloadLinkWrapper`, chromedp.ByID),

		// Step 9: Extract the download link text
		chromedp.AttributeValue(`#downloadLink`, "href", &downloadLink, nil),
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
		chromedp.Navigate("http://localhost:8080"),

		// Step 2: Enter an invalid YouTube URL
		chromedp.WaitVisible(`#url`, chromedp.ByID),
		chromedp.SendKeys(`#url`, "invalid-url", chromedp.ByID),

		// Step 3: Click the preview button
		chromedp.Click(`#previewButton`, chromedp.ByID),

		// Step 4: Check for an error message
		chromedp.WaitVisible(`.toast-error`, chromedp.ByQuery), // Assuming toastr shows errors
		chromedp.Text(`.toast-error`, &errorMessage, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("workflow error: %w", err)
	}

	if errorMessage != "Invalid YouTube URL" {
		return fmt.Errorf("unexpected error message: %s", errorMessage)
	}

	log.Printf("Invalid YouTube URL test passed")
	return nil
}

func testInvalidTimestamps(ctx context.Context) error {
	var errorMessage string

	err := chromedp.Run(ctx,
		// Step 1: Navigate to the app
		chromedp.Navigate("http://localhost:8080"),

		// Step 2: Enter a valid YouTube URL
		chromedp.SendKeys(`#url`, `https://www.youtube.com/watch?v=uyQMNagVi0I`, chromedp.ByID),
		chromedp.Click(`#previewButton`, chromedp.ByID),
		chromedp.WaitEnabled(`#formatSelect`, chromedp.ByID),
		chromedp.SetValue(`#formatSelect`, "mp4", chromedp.ByID),

		// Step 3: Enter invalid "From" and "To" values
		chromedp.SendKeys(`#from`, "invalid", chromedp.ByID),
		chromedp.SendKeys(`#to`, "invalid", chromedp.ByID),

		// Step 4: Attempt to clip
		chromedp.Click(`#clipButton`, chromedp.ByID),

		// Step 5: Check for error message
		chromedp.WaitVisible(`.toast-error`, chromedp.ByQuery),
		chromedp.Text(`.toast-error`, &errorMessage, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("workflow error: %w", err)
	}

	if errorMessage != "Invalid timestamps" {
		return fmt.Errorf("unexpected error message: %s", errorMessage)
	}

	log.Printf("Invalid timestamps test passed")
	return nil
}
