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
	if artifact.MetadataPath == "" || artifact.JobID != "job-0001" || artifact.TaskID != "go-test-root" {
		t.Fatalf("expected metadata-backed task report artifact, got %#v", artifact)
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

func TestListArtifactsSearchArchiveDeleteAndLineage(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	report, err := store.WriteTaskRunReport(TaskRunReport{
		ID:        "report",
		JobID:     "job-42",
		TaskID:    "task-42",
		Label:     "Nightly report",
		Command:   "go test ./...",
		Cwd:       ".",
		Source:    "package.json",
		Status:    "success",
		StartedAt: time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC),
		Message:   "done",
	})
	if err != nil {
		t.Fatalf("WriteTaskRunReport returned error: %v", err)
	}
	matches, err := store.ListArtifacts(ListOptions{Query: "nightly"})
	if err != nil {
		t.Fatalf("ListArtifacts returned error: %v", err)
	}
	if len(matches) != 1 || matches[0].JobID != "job-42" {
		t.Fatalf("unexpected artifact search matches: %#v", matches)
	}
	lineage, err := store.Lineage(report.RelPath)
	if err != nil {
		t.Fatalf("Lineage returned error: %v", err)
	}
	if len(lineage.Nodes) < 4 || len(lineage.Edges) < 3 {
		t.Fatalf("expected artifact/job/task/source lineage, got %#v", lineage)
	}
	archived, err := store.ArchiveArtifact(report.RelPath)
	if err != nil {
		t.Fatalf("ArchiveArtifact returned error: %v", err)
	}
	if !archived.Archived || !strings.Contains(archived.RelPath, "/archive/") {
		t.Fatalf("unexpected archived artifact: %#v", archived)
	}
	active, err := store.ListArtifacts(ListOptions{})
	if err != nil {
		t.Fatalf("ListArtifacts active returned error: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("archived artifact should be hidden by default: %#v", active)
	}
	all, err := store.ListArtifacts(ListOptions{IncludeArchived: true})
	if err != nil {
		t.Fatalf("ListArtifacts archived returned error: %v", err)
	}
	if len(all) != 1 || !all[0].Archived {
		t.Fatalf("expected archived artifact when included: %#v", all)
	}
	if err := store.DeleteArtifact(archived.RelPath); err != nil {
		t.Fatalf("DeleteArtifact returned error: %v", err)
	}
	all, err = store.ListArtifacts(ListOptions{IncludeArchived: true})
	if err != nil {
		t.Fatalf("ListArtifacts after delete returned error: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected delete to remove artifact and sidecar: %#v", all)
	}
}
