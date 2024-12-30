package utils

import (
	"fmt"
	"os/exec"
)

func CheckCommand(command string) error {
    _, err := exec.LookPath(command)
    if err != nil {
        return fmt.Errorf("%s is not installed or not in PATH", command)
    }
    return nil
}