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

func TestSlowWorkflowSpecsRequireDurableExplicitStarts(t *testing.T) {
	specs := SlowWorkflowSpecs()
	if len(specs) == 0 {
		t.Fatal("expected slow workflow specs")
	}
	for _, spec := range specs {
		if !spec.RequiresDurableJob || !spec.ProhibitedOnWorkspaceOpen || !spec.RequiresExplicitStart || !spec.Cancellable {
			t.Fatalf("slow workflow spec is missing production guardrails: %#v", spec)
		}
		if err := ValidateWorkflowStart(spec.Kind, StartOptions{}); err == nil {
			t.Fatalf("expected implicit %s start to be rejected", spec.Kind)
		}
		if err := ValidateWorkflowStart(spec.Kind, StartOptions{ExplicitUserStart: true}); err != nil {
			t.Fatalf("expected explicit %s start to be accepted: %v", spec.Kind, err)
		}
		if !RequiresDurableJob(spec.Kind) || !ProhibitedOnWorkspaceOpen(spec.Kind) {
			t.Fatalf("expected helpers to recognize %s", spec.Kind)
		}
	}
}

func TestStartWorkflowEnforcesExplicitUserStart(t *testing.T) {
	service := New()
	if _, _, err := service.StartWorkflow(KindOCRExtraction, "OCR", StartOptions{}); err == nil {
		t.Fatal("expected implicit OCR workflow to be rejected")
	}
	job, _, err := service.StartWorkflow(KindOCRExtraction, "OCR", StartOptions{ExplicitUserStart: true})
	if err != nil {
		t.Fatalf("expected explicit OCR workflow to start: %v", err)
	}
	if job.Kind != KindOCRExtraction || job.Status != StatusRunning {
		t.Fatalf("unexpected workflow job: %#v", job)
	}
}

func TestUnknownWorkflowKindKeepsCompatibility(t *testing.T) {
	if err := ValidateWorkflowStart("custom-short-job", StartOptions{}); err != nil {
		t.Fatalf("unknown job kinds should remain compatible: %v", err)
	}
	if RequiresDurableJob("custom-short-job") || ProhibitedOnWorkspaceOpen("custom-short-job") {
		t.Fatal("unknown job kind should not be classified as a slow workflow")
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
