package perf

import (
	"testing"
	"time"
)

func TestRecorderKeepsRecentTimings(t *testing.T) {
	recorder := NewRecorder(2)
	base := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)

	recorder.Record("first", base, base.Add(10*time.Millisecond), time.Second, "first detail")
	recorder.Record("second", base, base.Add(20*time.Millisecond), time.Second, "second detail")
	recorder.Record("third", base, base.Add(30*time.Millisecond), time.Second, "third detail")

	records := recorder.Snapshot(0)
	if len(records) != 2 {
		t.Fatalf("expected recorder retention to keep two records, got %#v", records)
	}
	if records[0].Name != "second" || records[1].Name != "third" {
		t.Fatalf("expected newest records in chronological order, got %#v", records)
	}
}

func TestRecorderMarksOverBudgetTimings(t *testing.T) {
	recorder := NewRecorder(4)
	started := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)

	record := recorder.Record(TimingWorkspaceOpen, started, started.Add(3*time.Second), WorkspaceOpenBudget, "opened workspace")

	if record.WithinBudget {
		t.Fatalf("expected timing to be over budget: %#v", record)
	}
	if record.Duration != 3*time.Second {
		t.Fatalf("expected duration to be preserved, got %s", record.Duration)
	}
}

func TestRecorderSnapshotIsACopy(t *testing.T) {
	recorder := NewRecorder(4)
	now := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
	recorder.Record("copy", now, now.Add(time.Millisecond), time.Second, "detail")

	snapshot := recorder.Snapshot(1)
	snapshot[0].Name = "mutated"

	next := recorder.Snapshot(1)
	if next[0].Name != "copy" {
		t.Fatalf("expected snapshot mutation not to affect recorder, got %#v", next)
	}
}
