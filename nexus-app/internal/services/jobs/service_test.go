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

func TestServicePersistsJobsWhenRepositoryAttached(t *testing.T) {
	repo := &fakeJobRepository{}
	service := NewWithRepository(repo)
	job, _ := service.Start("task", "go test")
	service.AppendLog(job.ID, "line")
	service.Finish(job.ID, StatusSuccess, "done", nil)

	if len(repo.saved) != 3 {
		t.Fatalf("expected start/log/finish saves, got %d", len(repo.saved))
	}
	if repo.saved[len(repo.saved)-1].Status != StatusSuccess {
		t.Fatalf("expected final persisted status, got %#v", repo.saved[len(repo.saved)-1])
	}
}

func TestServiceLoadsPersistedJobsAndContinuesIDs(t *testing.T) {
	repo := &fakeJobRepository{listed: []Job{{ID: "job-0007", Kind: "task", Label: "old", Status: StatusSuccess}}}
	service := NewWithRepository(repo)
	job, _ := service.Start("task", "new")
	if job.ID != "job-0008" {
		t.Fatalf("expected next persisted id, got %q", job.ID)
	}
	if jobs := service.List(); len(jobs) != 2 {
		t.Fatalf("expected loaded and new jobs, got %#v", jobs)
	}
}

type fakeJobRepository struct {
	listed []Job
	saved  []Job
}

func (r *fakeJobRepository) SaveJob(job Job) error {
	r.saved = append(r.saved, job)
	return nil
}

func (r *fakeJobRepository) ListJobs() ([]Job, error) {
	return append([]Job{}, r.listed...), nil
}
