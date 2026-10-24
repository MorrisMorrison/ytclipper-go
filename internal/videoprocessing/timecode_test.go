package videoprocessing

import (
	"testing"
)

func TestToSeconds(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		hasError bool
	}{
		{"00:00:30", 30, false},
		{"00:01:30", 90, false},
		{"01:00:00", 3600, false},
		{"1:30", 90, false},
		{"30", 30, false},
		{"invalid", 0, true},
		{"00:xx:30", 0, true},
		{"1:30:60", 0, true}, // Invalid seconds
		{"xx:00:00", 0, true},
		{"00:60:00", 0, true},
		{"60:00", 0, true},
		{"1:xx", 0, true},
		{"60", 0, true},
		{"1:00:00:00", 0, true},
		{"100:00:00", 360000, false},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, err := ToSeconds(test.input)
			if test.hasError {
				if err == nil {
					t.Fatalf("Expected error for input %s, but got none", test.input)
				}
			} else {
				if err != nil {
					t.Fatalf("Did not expect error for input %s, but got: %v", test.input, err)
				}
				if result != test.expected {
					t.Fatalf("For input %s, expected %d, but got %d", test.input, test.expected, result)
				}
			}
		})
	}
}
