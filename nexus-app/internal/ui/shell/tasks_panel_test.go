package shell

import (
	"strings"
	"testing"
	"time"

	tasksSvc "nexusdesk/internal/services/tasks"
)

func TestFormatTaskRunIncludesSummaryAndOutput(t *testing.T) {
	text := formatTaskRun(tasksSvc.RunResult{
		Task:     tasksSvc.Task{Label: "go test ./...", Command: "go test ./...", Cwd: "."},
		Status:   "success",
		ExitCode: 0,
		Stdout:   "ok fixture\n",
		Stderr:   "",
		Duration: 150 * time.Millisecond,
		Message:  `Task "go test ./..." completed.`,
	})

	for _, expected := range []string{
		`Task "go test ./..." completed.`,
		"Status: success",
		"Exit code: 0",
		"Command: go test ./...",
		"Stdout",
		"ok fixture",
		"Stderr",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected task output to contain %q, got:\n%s", expected, text)
		}
	}
}

func TestTaskRowsEmpty(t *testing.T) {
	rows := taskRows(nil, func(tasksSvc.Task) {})
	if len(rows) != 1 {
		t.Fatalf("expected one empty row, got %d", len(rows))
	}
}
