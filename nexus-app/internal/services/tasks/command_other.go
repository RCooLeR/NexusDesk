//go:build !windows

package tasks

import (
	"context"
	"os/exec"
)

func taskExecCommand(ctx context.Context, command string) *exec.Cmd {
	return exec.CommandContext(ctx, "sh", "-c", command)
}

func hideTaskCommandWindow(command *exec.Cmd) {}
