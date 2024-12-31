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

func TestExtractBitrate(t *testing.T) {
	tests := []struct {
		name       string
		additional string
		expected   string
	}{
		{
			name:       "Valid bitrate in KiB",
			additional: "128KiB",
			expected:   "128KiB",
		},
		{
			name:       "Valid bitrate in MiB",
			additional: "1.2MiB",
			expected:   "1.2MiB",
		},
		{
			name:       "Valid bitrate with tilde (~)",
			additional: "~192KiB",
			expected:   "~192KiB",
		},
		{
			name:       "No bitrate",
			additional: "Some other text",
			expected:   "N/A",
		},
		{
			name:       "Empty string",
			additional: "",
			expected:   "N/A",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := extractBitrate(test.additional)
			if result != test.expected {
				t.Errorf("For input '%s', expected '%s', but got '%s'", test.additional, test.expected, result)
			}
		})
	}
}
