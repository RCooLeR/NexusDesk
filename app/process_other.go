//go:build !windows

package main

import "os/exec"

func configureHiddenCommand(command *exec.Cmd) {
}
