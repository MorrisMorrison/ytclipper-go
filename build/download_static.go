package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	err = os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func main() {
	files := map[string]string{
		"https://cdnjs.cloudflare.com/ajax/libs/toastr.js/latest/toastr.min.js": "static/scripts/toastr.min.js",
		"https://cdnjs.cloudflare.com/ajax/libs/toastr.js/latest/toastr.min.css": "static/css/toastr.min.css",
		"https://cdnjs.cloudflare.com/ajax/libs/video.js/7.20.3/video.min.js":    "static/scripts/video.min.js",
		"https://cdnjs.cloudflare.com/ajax/libs/video.js/7.20.3/video-js.min.css": "static/css/video-js.min.css",
		"https://code.jquery.com/jquery.min.js": "static/scripts/jquery.min.js",
		"https://raw.githubusercontent.com/videojs/videojs-youtube/refs/heads/main/dist/Youtube.min.js": "static/scripts/Youtube.min.js",
	}

	for url, dest := range files {
		fmt.Printf("Downloading %s -> %s\n", url, dest)
		if err := downloadFile(url, dest); err != nil {
			fmt.Printf("Failed to download %s: %v\n", url, err)
		} else {
			fmt.Printf("Downloaded %s -> %s\n", url, dest)
		}
	}
}
