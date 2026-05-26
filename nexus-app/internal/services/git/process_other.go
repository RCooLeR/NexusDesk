//go:build !windows

package git

import "os/exec"

func hideGitCommandWindow(command *exec.Cmd) {}
