package videoprocessing

import (
	"testing"
)

func TestExtractDuration(t *testing.T) {
	output := "Duration: 00:01:30"
	expected := "00:01:30"

	duration := ExtractDuration(output)
	if duration != expected {
		t.Errorf("Expected %s, but got %s", expected, duration)
	}
}