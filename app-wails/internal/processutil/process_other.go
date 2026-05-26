//go:build !windows

package processutil

import "os/exec"

func ConfigureHiddenCommand(command *exec.Cmd) {
}
