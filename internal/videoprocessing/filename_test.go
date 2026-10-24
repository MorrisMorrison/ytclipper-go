package videoprocessing

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSanitizeFileName(t *testing.T) {
	tests := map[string]string{
		"Never Gonna Give You Up":      "Never Gonna Give You Up",
		"../../etc/passwd":             "etc passwd",
		`a\b/c:d*e?f"g<h>i|j'k` + "`l": "a b c d e f g h i j k l",
		"tab\tnew\nline\x00nul":        "tab new line nul",
		"rtl\u202eoverride":            "rtl override",
		"  lots   of   space  ":        "lots of space",
		"...dots...":                   "dots",
		"Ünïcödé 日本語 🎵":                "Ünïcödé 日本語 🎵",
		"":                             "",
		"///":                          "",
	}
	for in, want := range tests {
		if got := SanitizeFileName(in); got != want {
			t.Errorf("SanitizeFileName(%q) = %q; want %q", in, got, want)
		}
	}

	long := SanitizeFileName(strings.Repeat("é", 500))
	if n := utf8.RuneCountInString(long); n != maxFileNameBaseLength {
		t.Errorf("long name has %d runes; want %d", n, maxFileNameBaseLength)
	}
}

func TestDownloadName(t *testing.T) {
	tests := []struct {
		title, id, from, to, ext, want string
	}{
		{"Video title", "dQw4w9WgXcQ", "00:01:30", "00:02:00", ".mp4", "Video title [0-01-30 - 0-02-00].mp4"},
		{"Song: Live/Remix", "dQw4w9WgXcQ", "01:00:00", "12:34:56", ".mp3", "Song Live Remix [1-00-00 - 12-34-56].mp3"},
		{"", "dQw4w9WgXcQ", "00:00:00", "00:00:10", ".webm", "dQw4w9WgXcQ [0-00-00 - 0-00-10].webm"},
		{"???", "dQw4w9WgXcQ", "100:00:00", "100:00:01", ".mkv", "dQw4w9WgXcQ [100-00-00 - 100-00-01].mkv"},
	}
	for _, tt := range tests {
		if got := DownloadName(tt.title, tt.id, tt.from, tt.to, tt.ext); got != tt.want {
			t.Errorf("DownloadName(%q, ...) = %q; want %q", tt.title, got, tt.want)
		}
	}
}
