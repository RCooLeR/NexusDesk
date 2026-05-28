package shell

import (
	"strings"
	"testing"
	"time"

	jobsSvc "nexusdesk/internal/services/jobs"
	metadataSvc "nexusdesk/internal/services/metadata"
)

func TestJobSummaryIncludesMessageErrorAndTail(t *testing.T) {
	summary := jobSummary(jobsSvc.Job{
		Status:  jobsSvc.StatusFailed,
		Message: "Task failed.",
		Error:   "exit status 1",
		LogTail: []string{"older", "last line"},
	})
	for _, expected := range []string{"Task failed.", "exit status 1", "last line"} {
		if !strings.Contains(summary, expected) {
			t.Fatalf("expected job summary to contain %q, got %q", expected, summary)
		}
	}
}

func TestJobRowsEmpty(t *testing.T) {
	rows := jobRows(nil, func(string) {}, func(string) {}, func(string) {}, nil, nil)
	if len(rows) != 1 {
		t.Fatalf("expected one empty row, got %d", len(rows))
	}
}

func TestTaskRunOutputFormatting(t *testing.T) {
	output := formatTaskRunRecord(metadataSvc.TaskRunRecord{
		JobID:        "job-0001",
		TaskID:       "go-test",
		Command:      "go test ./...",
		Cwd:          ".",
		Status:       "success",
		ExitCode:     0,
		Stdout:       "ok\n",
		Stderr:       "",
		Message:      "Task completed.",
		ArtifactPath: ".nexusdesk/artifacts/task-runs/run.md",
		DurationMs:   42,
	})
	for _, expected := range []string{"Task completed.", "go test ./...", "Artifact: .nexusdesk/artifacts/task-runs/run.md", "ok"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q:\n%s", expected, output)
		}
	}
}

func TestTaskRunHasOutput(t *testing.T) {
	if !taskRunHasOutput(metadataSvc.TaskRunRecord{Stdout: "ok"}) {
		t.Fatal("expected stdout to count as output")
	}
	if taskRunHasOutput(metadataSvc.TaskRunRecord{}) {
		t.Fatal("empty task run should not have output")
	}
}

func TestArtifactRecordSortTimePrefersGenerated(t *testing.T) {
	generated := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
	created := generated.Add(-time.Hour)
	sorted := artifactRecordSortTime(metadataSvc.ArtifactRecord{
		GeneratedAt: generated,
		CreatedAt:   created,
	})
	if !sorted.Equal(generated) {
		t.Fatalf("expected generated time %v, got %v", generated, sorted)
	}
}

func TestJobHasOutput(t *testing.T) {
	if !jobHasOutput(jobsSvc.Job{Message: "running"}) {
		t.Fatal("expected message to count as job output")
	}
	if !jobHasOutput(jobsSvc.Job{LogTail: []string{"line"}}) {
		t.Fatal("expected log tail to count as job output")
	}
	if jobHasOutput(jobsSvc.Job{}) {
		t.Fatal("empty job should not have output")
	}
}

func TestFormatJobRecordIncludesStatusAndLogTail(t *testing.T) {
	started := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	completed := started.Add(3 * time.Second)
	output := formatJobRecord(jobsSvc.Job{
		ID:          "job-9",
		Kind:        "connector-query",
		Label:       "Connector query (Warehouse)",
		Status:      jobsSvc.StatusSuccess,
		Message:     "Query completed.",
		LogTail:     []string{"Rows: shown=50 total=50 duration=320ms"},
		StartedAt:   started,
		CompletedAt: completed,
	})
	for _, expected := range []string{"Query completed.", "Status: success", "Kind: connector-query", "Log tail", "Rows: shown=50 total=50 duration=320ms"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected job output to contain %q:\n%s", expected, output)
		}
	}
}
