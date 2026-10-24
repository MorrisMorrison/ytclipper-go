package api

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"ytclipper-go/utils"
)

var (
	videoIDPattern    = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	timeFormatPattern = regexp.MustCompile(`^\d{2,}:[0-5]\d:[0-5]\d$`)
	formatPattern     = regexp.MustCompile(`^\d+$`)
)

var youtubeHosts = map[string]bool{
	"youtube.com":       true,
	"www.youtube.com":   true,
	"m.youtube.com":     true,
	"music.youtube.com": true,
}

// extractVideoID returns the 11-character video ID of a YouTube link, accepting
// watch, shorts, live and youtu.be URLs with any extra query parameters.
func extractVideoID(rawURL string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", false
	}

	host := strings.ToLower(u.Hostname())
	var id string
	switch {
	case host == "youtu.be":
		id = strings.Trim(u.Path, "/")
	case youtubeHosts[host]:
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		switch {
		case len(parts) == 1 && parts[0] == "watch":
			id = u.Query().Get("v")
		case len(parts) == 2 && (parts[0] == "shorts" || parts[0] == "live"):
			id = parts[1]
		}
	}

	if !videoIDPattern.MatchString(id) {
		return "", false
	}
	return id, true
}

// canonicalYoutubeURL turns any accepted YouTube link into a plain watch URL,
// dropping playlist and tracking parameters before the URL reaches yt-dlp.
func canonicalYoutubeURL(rawURL string) (string, bool) {
	id, ok := extractVideoID(rawURL)
	if !ok {
		return "", false
	}
	return "https://www.youtube.com/watch?v=" + id, true
}

func isValidYoutubeUrl(rawURL string) bool {
	_, ok := extractVideoID(rawURL)
	return ok
}

func isValidTimeFormat(time string) bool {
	return timeFormatPattern.MatchString(time)
}

func isValidFormat(format string) bool {
	return formatPattern.MatchString(format)
}

// validateCreateClipDto checks the request and rewrites the URL to its
// canonical form.
func validateCreateClipDto(createClipDto *CreateClipDTO) error {
	canonical, ok := canonicalYoutubeURL(createClipDto.Url)
	if !ok {
		return errors.New("Invalid YouTube URL")
	}
	createClipDto.Url = canonical

	if !isValidTimeFormat(createClipDto.From) || !isValidTimeFormat(createClipDto.To) {
		return errors.New("Invalid time format. Use HH:MM:SS.")
	}

	from, _ := utils.ToSeconds(createClipDto.From)
	to, _ := utils.ToSeconds(createClipDto.To)
	if to <= from {
		return errors.New("End time must be after start time.")
	}

	if !isValidFormat(createClipDto.Format) {
		return errors.New("Invalid format. Must be a numeric value.")
	}

	return nil
}
