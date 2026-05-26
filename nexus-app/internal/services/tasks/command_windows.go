//go:build windows

package tasks

import (
	"context"
	"os/exec"
	"syscall"
)

func taskExecCommand(ctx context.Context, command string) *exec.Cmd {
	return exec.CommandContext(ctx, "cmd", "/C", command)
}

func hideTaskCommandWindow(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
