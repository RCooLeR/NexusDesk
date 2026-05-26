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

func runDiscovered(parent context.Context, root string, taskID string) (RunResult, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return RunResult{}, errors.New("open a workspace before running tasks")
	}
	if parent == nil {
		parent = context.Background()
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return RunResult{}, errors.New("task id is required")
	}
	summary, err := discover(root)
	if err != nil {
		return RunResult{}, err
	}
	selected, ok := findTask(summary.Tasks, taskID)
	if !ok {
		return RunResult{}, errors.New("task is no longer available in this workspace")
	}
	if err := validateRunnableTask(selected); err != nil {
		return RunResult{}, err
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return RunResult{}, err
	}
	cwd := filepath.Clean(filepath.Join(absRoot, filepath.FromSlash(selected.Cwd)))
	if selected.Cwd == "." || selected.Cwd == "" {
		cwd = absRoot
	}
	if err := ensureInsideRoot(absRoot, cwd); err != nil {
		return RunResult{}, err
	}

	started := time.Now().UTC()
	ctx, cancel := context.WithTimeout(parent, runTimeout)
	defer cancel()
	command := taskExecCommand(ctx, selected.Command)
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
	message := fmt.Sprintf("Task %q completed.", selected.Label)
	if ctx.Err() == context.DeadlineExceeded {
		exitCode = -1
		status = "timeout"
		message = fmt.Sprintf("Task %q timed out after %s.", selected.Label, runTimeout)
	} else if ctx.Err() == context.Canceled {
		exitCode = -1
		status = "canceled"
		message = fmt.Sprintf("Task %q was canceled.", selected.Label)
	} else if err != nil {
		status = "failed"
		message = fmt.Sprintf("Task %q failed.", selected.Label)
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
			message = err.Error()
		}
	}

	return RunResult{
		Task:        selected,
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

func findTask(tasks []Task, taskID string) (Task, bool) {
	for _, task := range tasks {
		if task.ID == taskID {
			return task, true
		}
	}
	return Task{}, false
}

func validateRunnableTask(task Task) error {
	switch task.Kind {
	case "npm-script":
		if strings.HasPrefix(task.Command, "npm run ") {
			return nil
		}
	case "go-test":
		if strings.HasPrefix(task.Command, "go test ") {
			return nil
		}
	case "compose":
		if strings.HasPrefix(task.Command, "docker compose -f ") && strings.HasSuffix(task.Command, " config") {
			return nil
		}
	}
	return fmt.Errorf("task %q is not runnable by the safe task runner", task.Label)
}

type limitWriter struct {
	buffer *bytes.Buffer
	limit  int
}

func (w limitWriter) Write(bytes []byte) (int, error) {
	if w.buffer.Len() < w.limit {
		remaining := w.limit - w.buffer.Len()
		if len(bytes) <= remaining {
			_, _ = w.buffer.Write(bytes)
		} else {
			_, _ = w.buffer.Write(bytes[:remaining])
			_, _ = w.buffer.WriteString("\n[output truncated]\n")
		}
	}
	return len(bytes), nil
}
