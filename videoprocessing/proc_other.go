//go:build !unix

package videoprocessing

import "os/exec"

// killProcessGroupOnCancel keeps exec's default (kill the process itself) on
// systems without Unix process groups; waitDelay still bounds how long a
// surviving child can block the caller. The app is deployed on Linux.
func killProcessGroupOnCancel(cmd *exec.Cmd) {}
