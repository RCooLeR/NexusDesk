package artifacts

import (
	"os"
	"path/filepath"
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

func TestListAndReadTaskRunReports(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.WriteTaskRunReport(TaskRunReport{
		ID:        "first",
		Label:     "First task",
		Command:   "go test ./...",
		Cwd:       ".",
		Status:    "success",
		StartedAt: time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC),
		Message:   "first done",
	})
	if err != nil {
		t.Fatalf("WriteTaskRunReport first returned error: %v", err)
	}
	second, err := store.WriteTaskRunReport(TaskRunReport{
		ID:        "second",
		Label:     "Second task",
		Command:   "npm test",
		Cwd:       ".",
		Status:    "failed",
		StartedAt: time.Date(2026, 5, 27, 12, 0, 1, 0, time.UTC),
		Message:   "second done",
	})
	if err != nil {
		t.Fatalf("WriteTaskRunReport second returned error: %v", err)
	}

	reports, err := store.ListTaskRunReports()
	if err != nil {
		t.Fatalf("ListTaskRunReports returned error: %v", err)
	}
	if len(reports) != 2 || reports[0].RelPath != second.RelPath || reports[1].RelPath != first.RelPath {
		t.Fatalf("unexpected report order: %#v", reports)
	}
	text, err := store.ReadArtifactText(second.RelPath)
	if err != nil {
		t.Fatalf("ReadArtifactText returned error: %v", err)
	}
	if !strings.Contains(text, "Second task") || !strings.Contains(text, "npm test") {
		t.Fatalf("unexpected artifact text: %s", text)
	}

	outside := filepath.ToSlash(filepath.Join("..", "outside.md"))
	if _, err := store.ReadArtifactText(outside); err == nil {
		t.Fatal("expected traversal read to fail")
	}
}
