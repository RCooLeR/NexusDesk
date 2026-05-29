package tasks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	terminalDefaultTimeout = 30 * time.Second
	terminalMaxTimeout     = 2 * time.Minute
)

func runTerminalCommand(parent context.Context, root string, request TerminalRequest) (TerminalResult, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return TerminalResult{}, errors.New("open a workspace before running terminal commands")
	}
	if parent == nil {
		parent = context.Background()
	}
	commandName := strings.TrimSpace(request.Command)
	if err := validateTerminalCommand(commandName); err != nil {
		return TerminalResult{}, err
	}
	args := sanitizeTerminalArgs(request.Args)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return TerminalResult{}, err
	}
	cwd := absRoot
	if strings.TrimSpace(request.Cwd) != "" && strings.TrimSpace(request.Cwd) != "." {
		cwd = filepath.Clean(filepath.Join(absRoot, filepath.FromSlash(strings.TrimSpace(request.Cwd))))
	}
	if err := ensureInsideRoot(absRoot, cwd); err != nil {
		return TerminalResult{}, err
	}
	timeout := terminalTimeout(request.TimeoutSeconds)
	started := time.Now().UTC()
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	command := taskExecCommand(ctx, commandName, args...)
	command.Dir = cwd
	hideTaskCommandWindow(command)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = limitWriter{buffer: &stdout, limit: outputLimit}
	command.Stderr = limitWriter{buffer: &stderr, limit: outputLimit}
	err = command.Run()
	completed := time.Now().UTC()

	exitCode := 0
	status := "success"
	message := fmt.Sprintf("Terminal command %q completed.", commandName)
	if ctx.Err() == context.DeadlineExceeded {
		exitCode = -1
		status = "timeout"
		message = fmt.Sprintf("Terminal command %q timed out after %s.", commandName, timeout)
	} else if ctx.Err() == context.Canceled {
		exitCode = -1
		status = "canceled"
		message = fmt.Sprintf("Terminal command %q was canceled.", commandName)
	} else if err != nil {
		status = "failed"
		message = fmt.Sprintf("Terminal command %q failed.", commandName)
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
			message = err.Error()
		}
	}

	return TerminalResult{
		Command:     commandName,
		Args:        append([]string{}, args...),
		Cwd:         filepath.ToSlash(mustRelDir(absRoot, cwd)),
		Status:      status,
		ExitCode:    exitCode,
		Stdout:      stdout.String(),
		Stderr:      stderr.String(),
		StartedAt:   started,
		CompletedAt: completed,
		Duration:    completed.Sub(started),
		Message:     message,
	}, nil
}

func validateTerminalCommand(command string) error {
	if command == "" {
		return errors.New("terminal command is required")
	}
	if strings.ContainsAny(command, `/\`) {
		return errors.New("terminal command must be a command name on PATH, not a path")
	}
	normalized := strings.TrimSuffix(strings.ToLower(command), ".exe")
	switch normalized {
	case "sh", "bash", "zsh", "fish", "cmd", "powershell", "pwsh":
		return errors.New("shell interpreters are blocked; provide a command plus explicit args instead")
	}
	return nil
}

func sanitizeTerminalArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		out = append(out, arg)
	}
	return out
}

func terminalTimeout(seconds int) time.Duration {
	if seconds <= 0 {
		return terminalDefaultTimeout
	}
	timeout := time.Duration(seconds) * time.Second
	if timeout > terminalMaxTimeout {
		return terminalMaxTimeout
	}
	return timeout
}

func mustRelDir(root string, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return "."
	}
	return rel
}
