package api

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"ytclipper-go/config"
	"ytclipper-go/utils"
	"ytclipper-go/videoprocessing"
)

var (
	timeFormatPattern = regexp.MustCompile(`^\d{1,3}:[0-5]\d:[0-5]\d$`)
	formatPattern     = regexp.MustCompile(`^\d{1,6}(\+\d{1,6})?$`)
)

var youtubeHosts = map[string]bool{
	"youtube.com":       true,
	"www.youtube.com":   true,
	"m.youtube.com":     true,
	"music.youtube.com": true,
}

// extractVideoID returns the 11-character video ID of a YouTube link, accepting
// watch, shorts, live and youtu.be URLs with any extra query parameters.
// Only the ID reaches yt-dlp (as a plain watch URL), so playlist and tracking
// parameters are dropped.
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

	if !videoprocessing.IsValidVideoID(id) {
		return "", false
	}
	return id, true
}

func isValidTimeFormat(time string) bool {
	return timeFormatPattern.MatchString(time)
}

// parseTime validates an "H:MM:SS" timestamp (1 to 3 hour digits) and returns
// it in seconds and as zero-padded "HH:MM:SS".
func parseTime(time string) (int, string, bool) {
	if !isValidTimeFormat(time) {
		return 0, "", false
	}
	seconds, err := utils.ToSeconds(time)
	if err != nil {
		return 0, "", false
	}
	return seconds, fmt.Sprintf("%02d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60), true
}

func isValidFormat(format string) bool {
	return formatPattern.MatchString(format)
}

// validateCreateClipDto checks the request and turns it into a clip job
// request for the video it names.
func validateCreateClipDto(createClipDto *CreateClipDTO) (videoprocessing.ClipRequest, error) {
	videoID, ok := extractVideoID(createClipDto.Url)
	if !ok {
		return videoprocessing.ClipRequest{}, errors.New("Invalid YouTube URL")
	}

	from, fromHHMMSS, fromOK := parseTime(createClipDto.From)
	to, toHHMMSS, toOK := parseTime(createClipDto.To)
	if !fromOK || !toOK {
		return videoprocessing.ClipRequest{}, errors.New("Invalid time format. Use HH:MM:SS.")
	}
	if to <= from {
		return videoprocessing.ClipRequest{}, errors.New("End time must be after start time.")
	}
	if maxMinutes := config.CONFIG.MaxClipLengthInMinutes; maxMinutes > 0 && to-from > maxMinutes*60 {
		return videoprocessing.ClipRequest{}, fmt.Errorf("Clip is too long (max %d minutes).", maxMinutes)
	}

	preset, format := strings.TrimSpace(createClipDto.Preset), strings.TrimSpace(createClipDto.Format)
	switch {
	case preset != "" && format != "":
		return videoprocessing.ClipRequest{}, errors.New("Specify either a preset or a format, not both.")
	case preset == "" && format == "":
		return videoprocessing.ClipRequest{}, errors.New("Specify a preset (video or audio) or a format.")
	case preset != "" && preset != videoprocessing.PresetVideo && preset != videoprocessing.PresetAudio:
		return videoprocessing.ClipRequest{}, errors.New(`Invalid preset. Use "video" or "audio".`)
	case format != "" && !isValidFormat(format):
		return videoprocessing.ClipRequest{}, errors.New("Invalid format. Use a format id like 137 or 137+140.")
	}

	return videoprocessing.ClipRequest{
		VideoID: videoID,
		From:    fromHHMMSS,
		To:      toHHMMSS,
		Preset:  preset,
		Format:  format,
	}, nil
}
