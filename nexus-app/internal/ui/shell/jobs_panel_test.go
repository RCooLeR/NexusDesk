package shell

import (
	"strings"
	"testing"

	jobsSvc "nexusdesk/internal/services/jobs"
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
	rows := jobRows(nil, func(string) {})
	if len(rows) != 1 {
		t.Fatalf("expected one empty row, got %d", len(rows))
	}
}
