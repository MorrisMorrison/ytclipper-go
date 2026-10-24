package api

import (
	"testing"
	"ytclipper-go/config"
)

func TestExtractVideoID(t *testing.T) {
	const want = "dQw4w9WgXcQ"
	tests := []struct {
		url     string
		isValid bool
	}{
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"http://youtu.be/dQw4w9WgXcQ", true},
		{"https://youtu.be/dQw4w9WgXcQ?si=abc123", true},
		{"https://youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"https://m.youtube.com/watch?v=dQw4w9WgXcQ&t=42s", true},
		{"https://music.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PL123&index=2", true},
		{"https://www.youtube.com/shorts/dQw4w9WgXcQ", true},
		{"https://www.youtube.com/live/dQw4w9WgXcQ?feature=share", true},
		{"  https://www.youtube.com/watch?v=dQw4w9WgXcQ  ", true},
		{"https://www.youtube.com/embed/dQw4w9WgXcQ", false},
		{"https://www.youtu.be.com/watch?v=dQw4w9WgXcQ", false},
		{"https://evil.com/watch?v=dQw4w9WgXcQ", false},
		{"https://vimeo.com/123456", false},
		{"invalidurl", false},
		{"https://youtube.com/watch?", false},
		{"https://www.youtube.com/watch?v=short", false},
		{"ftp://www.youtube.com/watch?v=dQw4w9WgXcQ", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got, ok := extractVideoID(tt.url)
			if ok != tt.isValid {
				t.Fatalf("extractVideoID(%q) ok = %v; want %v", tt.url, ok, tt.isValid)
			}
			if ok && got != want {
				t.Errorf("extractVideoID(%q) = %q; want %q", tt.url, got, want)
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
		{"24:00:00", true},  // Streams can be longer than a day
		{"100:00:00", true}, // ...much longer
		{"999:59:59", true},
		{"1:02:03", true}, // one hour digit
		{"1000:00:00", false},
		{"0000000000000000000001:00:00", false}, // would overflow ToSeconds
		{"12:60:00", false},
		{"12:34:60", false},
		{"1:2:3", false},
		{":00:00", false},
		{"invalid", false},
		{"12:34", false},
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
		{"001", true},
		{"1a", false},
		{"", false},
		{"-123", false},
		{"137+140", true},
		{"137+", false},
		{"+140", false},
		{"1+2+3", false},
		{"bestvideo", false},
		{"137 140", false},
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
	original := config.CONFIG.MaxClipLengthInMinutes
	config.CONFIG.MaxClipLengthInMinutes = 30
	t.Cleanup(func() { config.CONFIG.MaxClipLengthInMinutes = original })

	valid := func() *CreateClipDTO {
		return &CreateClipDTO{
			Url:    "https://youtu.be/dQw4w9WgXcQ?si=x",
			From:   "00:01:30",
			To:     "00:02:30",
			Format: "399",
		}
	}

	tests := []struct {
		name        string
		mutate      func(d *CreateClipDTO)
		expectedMsg string
	}{
		{"Valid format", func(d *CreateClipDTO) {}, ""},
		{"Valid merged format", func(d *CreateClipDTO) { d.Format = "137+140" }, ""},
		{"Valid video preset", func(d *CreateClipDTO) { d.Format, d.Preset = "", "video" }, ""},
		{"Valid audio preset", func(d *CreateClipDTO) { d.Format, d.Preset = "", "audio" }, ""},
		{"Invalid URL", func(d *CreateClipDTO) { d.Url = "https://invalidurl.com" }, "Invalid YouTube URL"},
		{"Invalid Time", func(d *CreateClipDTO) { d.From = "invalid" }, "Invalid time format. Use HH:MM:SS."},
		{"End before start", func(d *CreateClipDTO) { d.From, d.To = "00:02:30", "00:01:30" }, "End time must be after start time."},
		{"End equals start", func(d *CreateClipDTO) { d.To = d.From }, "End time must be after start time."},
		{"Invalid Format", func(d *CreateClipDTO) { d.Format = "invalid" }, "Invalid format. Use a format id like 137 or 137+140."},
		{"Format selector", func(d *CreateClipDTO) { d.Format = "bv*+ba" }, "Invalid format. Use a format id like 137 or 137+140."},
		{"Preset and format", func(d *CreateClipDTO) { d.Preset = "video" }, "Specify either a preset or a format, not both."},
		{"Neither", func(d *CreateClipDTO) { d.Format = "" }, "Specify a preset (video or audio) or a format."},
		{"Unknown preset", func(d *CreateClipDTO) { d.Format, d.Preset = "", "4k" }, `Invalid preset. Use "video" or "audio".`},
		{"Invalid end time", func(d *CreateClipDTO) { d.To = "00:02:61" }, "Invalid time format. Use HH:MM:SS."},
		{"Max length", func(d *CreateClipDTO) { d.From, d.To = "01:00:00", "01:30:00" }, ""},
		{"Too long", func(d *CreateClipDTO) { d.From, d.To = "01:00:00", "01:30:01" }, "Clip is too long (max 30 minutes)."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dto := valid()
			tt.mutate(dto)
			req, err := validateCreateClipDto(dto)
			if tt.expectedMsg == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if req.VideoID != "dQw4w9WgXcQ" || req.From != dto.From || req.To != dto.To {
					t.Errorf("unexpected request %+v", req)
				}
				if req.Preset != dto.Preset || req.Format != dto.Format {
					t.Errorf("preset/format not carried over: %+v", req)
				}
				return
			}
			if err == nil || err.Error() != tt.expectedMsg {
				t.Errorf("Expected error %q, got %v", tt.expectedMsg, err)
			}
		})
	}
}

func TestValidateCreateClipDtoNormalizesTimes(t *testing.T) {
	req, err := validateCreateClipDto(&CreateClipDTO{Url: "https://youtu.be/dQw4w9WgXcQ", From: "1:02:03", To: "1:02:13", Preset: "audio"})
	if err != nil {
		t.Fatal(err)
	}
	if req.From != "01:02:03" || req.To != "01:02:13" {
		t.Errorf("times = %q - %q; want 01:02:03 - 01:02:13", req.From, req.To)
	}
}

func TestValidateCreateClipDtoWithoutLengthLimit(t *testing.T) {
	original := config.CONFIG.MaxClipLengthInMinutes
	config.CONFIG.MaxClipLengthInMinutes = 0
	t.Cleanup(func() { config.CONFIG.MaxClipLengthInMinutes = original })

	if _, err := validateCreateClipDto(&CreateClipDTO{Url: "https://youtu.be/dQw4w9WgXcQ", From: "00:00:00", To: "10:00:00", Preset: "video"}); err != nil {
		t.Errorf("0 must disable the limit: %v", err)
	}
}
