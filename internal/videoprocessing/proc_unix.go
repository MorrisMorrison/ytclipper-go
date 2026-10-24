//go:build unix

package videoprocessing

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// killProcessGroupOnCancel starts cmd in its own process group and makes
// cancelling its context kill the whole group, not only cmd itself. yt-dlp
// runs ffmpeg as a child for --download-sections; killing yt-dlp alone would
// leave ffmpeg running and holding the output pipes.
func killProcessGroupOnCancel(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
}
