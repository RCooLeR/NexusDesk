package metadata

import (
	"os"
	"testing"
	"time"

	jobssvc "nexusdesk/internal/services/jobs"
)

func TestEnsureCreatesSQLiteStoreAndManifest(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	status, err := store.Ensure()
	if err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}
	if status.SchemaVersion != 1 || status.SchemaHash == "" {
		t.Fatalf("unexpected status: %#v", status)
	}
	if _, err := os.Stat(status.Path); err != nil {
		t.Fatalf("expected sqlite db: %v", err)
	}
	if _, err := os.Stat(status.SchemaPath); err != nil {
		t.Fatalf("expected schema file: %v", err)
	}
	if len(status.Tables) < 3 {
		t.Fatalf("expected core tables, got %#v", status.Tables)
	}
}

func TestSaveAndListJobs(t *testing.T) {
	store := mustStore(t)
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	completed := started.Add(time.Second)
	err := store.SaveJob(jobssvc.Job{
		ID:          "job-0001",
		Kind:        "task",
		Label:       "go test ./...",
		Status:      jobssvc.StatusSuccess,
		Message:     "done",
		LogTail:     []string{"line 1", "line 2"},
		StartedAt:   started,
		CompletedAt: completed,
	})
	if err != nil {
		t.Fatalf("SaveJob returned error: %v", err)
	}
	jobs, err := store.ListJobs()
	if err != nil {
		t.Fatalf("ListJobs returned error: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != "job-0001" || len(jobs[0].LogTail) != 2 {
		t.Fatalf("unexpected jobs: %#v", jobs)
	}
}

func TestSaveAndListTaskRuns(t *testing.T) {
	store := mustStore(t)
	started := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	record := TaskRunRecord{
		JobID:       "job-0001",
		TaskID:      "go-test-root",
		Kind:        "go-test",
		Label:       "go test ./...",
		Command:     "go test ./...",
		Cwd:         ".",
		Status:      "success",
		ExitCode:    0,
		Stdout:      "ok\n",
		Message:     "done",
		StartedAt:   started,
		CompletedAt: started.Add(time.Second),
		DurationMs:  1000,
	}
	if err := store.SaveTaskRun(record); err != nil {
		t.Fatalf("SaveTaskRun returned error: %v", err)
	}
	runs, err := store.ListTaskRuns(10)
	if err != nil {
		t.Fatalf("ListTaskRuns returned error: %v", err)
	}
	if len(runs) != 1 || runs[0].TaskID != "go-test-root" || runs[0].Stdout != "ok\n" {
		t.Fatalf("unexpected task runs: %#v", runs)
	}
}

func mustStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Ensure(); err != nil {
		t.Fatal(err)
	}
	return store
}
