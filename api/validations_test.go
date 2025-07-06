package api

import (
	"testing"
)

func TestIsValidYoutubeUrl(t *testing.T) {
	tests := []struct {
		url     string
		isValid bool
	}{
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"http://youtu.be/dQw4w9WgXcQ", true},
		{"https://youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"https://www.youtube.com/embed/dQw4w9WgXcQ", false},    // Invalid embed URL
		{"https://www.youtu.be.com/watch?v=dQw4w9WgXcQ", false}, // Typo in domain
		{"https://vimeo.com/123456", false},                     // Non-YouTube URL
		{"invalidurl", false},                                   // Completely invalid URL
		{"https://youtube.com/watch?", false},                   // Missing video ID
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := isValidYoutubeUrl(tt.url); got != tt.isValid {
				t.Errorf("isValidYoutubeUrl(%q) = %v; want %v", tt.url, got, tt.isValid)
			}
		})
	}
}

func TestIsValidTimeFormat(t *testing.T) {
	tests := []struct {
		time    string
		isValid bool
	}{
		{"00:00:00", true},
		{"12:34:56", true},
		{"23:59:59", true},
		{"24:00:00", false}, // Out of range hour
		{"12:60:00", false}, // Out of range minutes
		{"12:34:60", false}, // Out of range seconds
		{"1:2:3", false},    // Missing leading zeros
		{"invalid", false},  // Completely invalid input
		{"12:34", false},    // Missing seconds
	}

	for _, tt := range tests {
		t.Run(tt.time, func(t *testing.T) {
			if got := isValidTimeFormat(tt.time); got != tt.isValid {
				t.Errorf("isValidTimeFormat(%q) = %v; want %v", tt.time, got, tt.isValid)
			}
		})
	}
}

func TestIsValidFormat(t *testing.T) {
	tests := []struct {
		format  string
		isValid bool
	}{
		{"399", true},
		{"22", true},
		{"1", true},
		{"001", true},   // Leading zeros should still be valid
		{"1a", false},   // Contains letters
		{"", false},     // Empty string
		{"-123", false}, // Negative number
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			if got := isValidFormat(tt.format); got != tt.isValid {
				t.Errorf("isValidFormat(%q) = %v; want %v", tt.format, got, tt.isValid)
			}
		})
	}
}

func TestValidateCreateClipDto(t *testing.T) {
	validDto := &CreateClipDTO{
		Url:    "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		From:   "00:01:30",
		To:     "00:02:30",
		Format: "399",
	}

	invalidUrlDto := &CreateClipDTO{
		Url:    "https://invalidurl.com",
		From:   "00:01:30",
		To:     "00:02:30",
		Format: "399",
	}

	invalidTimeDto := &CreateClipDTO{
		Url:    "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		From:   "invalid",
		To:     "00:02:30",
		Format: "399",
	}

	invalidFormatDto := &CreateClipDTO{
		Url:    "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		From:   "00:01:30",
		To:     "00:02:30",
		Format: "invalid",
	}

	tests := []struct {
		name        string
		dto         *CreateClipDTO
		expectErr   bool
		expectedMsg string
	}{
		{"Valid DTO", validDto, false, ""},
		{"Invalid URL", invalidUrlDto, true, "Invalid YouTube URL"},
		{"Invalid Time", invalidTimeDto, true, "Invalid time format. Use HH:MM:SS."},
		{"Invalid Format", invalidFormatDto, true, "Invalid format. Must be a numeric value."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateClipDto(tt.dto)
			if (err != nil) != tt.expectErr {
				t.Errorf("validateCreateClipDto() error = %v, expectErr %v", err, tt.expectErr)
			}
			if err != nil && err.Error() != tt.expectedMsg {
				t.Errorf("Expected error message %q, got %q", tt.expectedMsg, err.Error())
			}
		})
	}
}
