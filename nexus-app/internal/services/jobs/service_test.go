package jobs

import (
	"errors"
	"testing"
)

func TestServiceTracksJobLifecycle(t *testing.T) {
	service := New()
	job, _ := service.Start("task", "go test ./...")
	if job.ID == "" || job.Status != StatusRunning {
		t.Fatalf("unexpected started job: %#v", job)
	}
	service.AppendLog(job.ID, "line 1")
	service.Finish(job.ID, StatusSuccess, "done", nil)

	got, ok := service.Get(job.ID)
	if !ok {
		t.Fatal("expected job to be available")
	}
	if got.Status != StatusSuccess || got.Message != "done" || len(got.LogTail) != 1 {
		t.Fatalf("unexpected finished job: %#v", got)
	}
	if got.CompletedAt.IsZero() {
		t.Fatalf("expected completion timestamp: %#v", got)
	}
}

func TestServiceCancelPreservesCanceledStatus(t *testing.T) {
	service := New()
	job, ctx := service.Start("task", "slow")
	if !service.Cancel(job.ID) {
		t.Fatal("expected cancel to succeed")
	}
	if ctx.Err() == nil {
		t.Fatal("expected job context to be canceled")
	}
	service.Finish(job.ID, StatusFailed, "failed", errors.New("late failure"))
	got, _ := service.Get(job.ID)
	if got.Status != StatusCanceled || got.Message != "Cancel requested." {
		t.Fatalf("expected cancel state to be preserved, got %#v", got)
	}
}

func TestServiceCapsLogTail(t *testing.T) {
	service := New()
	job, _ := service.Start("task", "noisy")
	for index := 0; index < maxLogLines+4; index++ {
		service.AppendLog(job.ID, "line")
	}
	got, _ := service.Get(job.ID)
	if len(got.LogTail) != maxLogLines {
		t.Fatalf("expected capped log tail, got %d", len(got.LogTail))
	}
}
