//go:build windows

package tasks

import (
	"context"
	"os/exec"
	"syscall"
)

func taskExecCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func hideTaskCommandWindow(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
