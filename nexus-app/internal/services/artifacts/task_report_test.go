package artifacts

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestWriteTaskRunReportCreatesMarkdownArtifact(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	record := TaskRunReport{
		ID:          "abc123",
		JobID:       "job-0001",
		TaskID:      "go-test-root",
		Kind:        "go-test",
		Label:       "go test ./...",
		Command:     "go test ./...",
		Cwd:         ".",
		Status:      "success",
		ExitCode:    0,
		Stdout:      "ok fixture\n",
		Message:     "done",
		StartedAt:   time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC),
		CompletedAt: time.Date(2026, 5, 27, 12, 0, 1, 0, time.UTC),
		DurationMs:  1000,
	}
	artifact, err := store.WriteTaskRunReport(record)
	if err != nil {
		t.Fatalf("WriteTaskRunReport returned error: %v", err)
	}
	if !strings.HasPrefix(artifact.RelPath, ".nexusdesk/artifacts/task-runs/") {
		t.Fatalf("unexpected artifact path: %q", artifact.RelPath)
	}
	data, err := os.ReadFile(artifact.AbsPath)
	if err != nil {
		t.Fatalf("expected report file: %v", err)
	}
	text := string(data)
	for _, expected := range []string{"# Task Run Report", "go test ./...", "Status:** success", "ok fixture"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected report to contain %q, got:\n%s", expected, text)
		}
	}
}

func TestSafeNameFallsBackForEmptyInput(t *testing.T) {
	if got := safeName(" ??? "); got != "task-run" {
		t.Fatalf("safeName fallback = %q", got)
	}
}
