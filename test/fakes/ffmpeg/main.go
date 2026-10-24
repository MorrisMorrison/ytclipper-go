// Command ffmpeg is a stand-in for ffmpeg in the e2e suite (test/e2e). The
// app only checks at startup that ffmpeg is on PATH; the fake yt-dlp never
// calls it.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("ffmpeg version ytclipper-fake")
	if len(os.Args) > 1 && (os.Args[1] == "-version" || os.Args[1] == "--version") {
		return
	}
	fmt.Fprintln(os.Stderr, "ffmpeg (fake): not implemented")
	os.Exit(1)
}
