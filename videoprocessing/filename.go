package videoprocessing

import (
	"fmt"
	"strings"
	"unicode"
	"ytclipper-go/utils"
)

const maxFileNameBaseLength = 120 // runes, before the time range and extension

// fileNameReserved are characters that are unsafe or invalid in file names
// on common systems, plus quotes.
const fileNameReserved = "/\\:*?\"'<>|`"

// DownloadName builds the readable name a clip is downloaded as:
// "<title> [h-mm-ss - h-mm-ss].<ext>". Without a usable title it uses the
// video ID. ext includes the leading dot.
func DownloadName(title string, videoID string, from string, to string, ext string) string {
	base := SanitizeFileName(title)
	if base == "" {
		base = SanitizeFileName(videoID)
	}
	if base == "" {
		base = "clip"
	}
	if ext != "" {
		ext = "." + SanitizeFileName(strings.TrimPrefix(ext, "."))
	}
	return fmt.Sprintf("%s [%s - %s]%s", base, fileTimestamp(from), fileTimestamp(to), ext)
}

// fileTimestamp turns "00:01:30" into "0-01-30" (colons are not allowed in
// file names on Windows).
func fileTimestamp(hhmmss string) string {
	seconds, err := utils.ToSeconds(hhmmss)
	if err != nil {
		return SanitizeFileName(strings.ReplaceAll(hhmmss, ":", "-"))
	}
	return fmt.Sprintf("%d-%02d-%02d", seconds/3600, seconds/60%60, seconds%60)
}

// SanitizeFileName makes untrusted text (a video title) safe to use as a file
// name: no path separators, reserved characters, quotes, control or
// formatting characters; whitespace collapsed and length capped.
func SanitizeFileName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if !unicode.IsPrint(r) || strings.ContainsRune(fileNameReserved, r) {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}

	cleaned := strings.Join(strings.Fields(b.String()), " ")
	if runes := []rune(cleaned); len(runes) > maxFileNameBaseLength {
		cleaned = string(runes[:maxFileNameBaseLength])
	}
	return strings.Trim(cleaned, " .")
}
