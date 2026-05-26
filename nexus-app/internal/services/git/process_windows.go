//go:build windows

package git

import (
	"os/exec"
	"syscall"
)

func hideGitCommandWindow(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
