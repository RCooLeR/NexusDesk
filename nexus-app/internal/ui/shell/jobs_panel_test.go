package shell

import (
	"strings"
	"testing"

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
	rows := jobRows(nil, func(string) {}, func(string) {}, func(string) {}, nil)
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
