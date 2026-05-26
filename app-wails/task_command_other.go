//go:build !windows

package main

import (
	"context"
	"os/exec"
)

func taskExecCommand(ctx context.Context, command string) *exec.Cmd {
	return exec.CommandContext(ctx, "sh", "-c", command)
}
