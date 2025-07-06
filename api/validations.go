package api

import (
	"fmt"
	"regexp"
)

func isValidYoutubeUrl(url string) bool {
	regex := regexp.MustCompile(`http(?:s?):\/\/(?:www\.)?youtu(?:be\.com\/watch\?v=|\.be\/)([\w\-\_]*)(&(amp;)?‌​[\w\?‌​=]*)?`)
	return regex.MatchString(url)
}

func isValidTimeFormat(time string) bool {
	regex := regexp.MustCompile(`^(?:[0-1]?[0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]$`)
	return regex.MatchString(time)
}

func isValidFormat(format string) bool {
	regex := regexp.MustCompile(`^\d+$`)
	return regex.MatchString(format)
}

func validateCreateClipDto(createClipDto *CreateClipDTO) error {
	if !isValidYoutubeUrl(createClipDto.Url) {
		return fmt.Errorf("Invalid YouTube URL")
	}

	if !isValidTimeFormat(createClipDto.From) || !isValidTimeFormat(createClipDto.To) {
		return fmt.Errorf("Invalid time format. Use HH:MM:SS.")
	}

	if !isValidFormat(createClipDto.Format) {
		return fmt.Errorf("Invalid format. Must be a numeric value.")
	}

	return nil
}
