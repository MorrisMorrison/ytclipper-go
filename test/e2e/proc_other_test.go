//go:build e2e && !unix

package e2e

import "os/exec"

func setProcessGroup(*exec.Cmd) {}

// terminate can't send a graceful signal here, so e2e coverage
// (E2E_COVERDIR) is only collected on unix.
func terminate(*exec.Cmd) {}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
