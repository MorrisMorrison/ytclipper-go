package utils

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func CheckCommand(command string) error {
    _, err := exec.LookPath(command)
    if err != nil {
        return fmt.Errorf("%s is not installed or not in PATH", command)
    }
    return nil
}

func ToSeconds(duration string) (int, error) {
	parts := strings.Split(duration, ":")
	var totalSeconds int

	if len(parts) == 3 {
		hours, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("invalid hours value: %v", err)
		}
		minutes, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid minutes value: %v", err)
		}
		seconds, err := strconv.Atoi(parts[2])
		if err != nil {
			return 0, fmt.Errorf("invalid seconds value: %v", err)
		}
		totalSeconds = hours*3600 + minutes*60 + seconds
	} else if len(parts) == 2 {
		minutes, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, fmt.Errorf("invalid minutes value: %v", err)
		}
		seconds, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid seconds value: %v", err)
		}
		totalSeconds = minutes*60 + seconds
	} else if len(parts) == 1 {
		seconds, err := strconv.Atoi(parts[0]) 
		if err != nil {
			return 0, fmt.Errorf("invalid seconds value: %v", err)
		}
		totalSeconds = seconds
	} else {
		return 0, errors.New("invalid duration format: must be HH:MM:SS, MM:SS, or SS")
	}

	return totalSeconds, nil
}

