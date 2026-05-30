package jobs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestServicePersistsFullJobLogsUnderWorkspace(t *testing.T) {
	root := t.TempDir()
	service := New()
	if err := service.SetLogRoot(root); err != nil {
		t.Fatalf("SetLogRoot returned error: %v", err)
	}
	job, _ := service.Start("task", "noisy")
	for index := 0; index < maxLogLines+4; index++ {
		service.AppendLog(job.ID, "line "+time.Unix(int64(index), 0).UTC().Format(time.RFC3339))
	}
	got, _ := service.Get(job.ID)
	if len(got.LogTail) != maxLogLines || strings.Contains(strings.Join(got.LogTail, "\n"), "1970-01-01T00:00:00Z") {
		t.Fatalf("expected visible tail to keep newest %d lines, got %#v", maxLogLines, got.LogTail)
	}
	if got.LogPath == "" || filepath.Base(filepath.Dir(got.LogPath)) != job.ID {
		t.Fatalf("expected job log path under job id directory, got %#v", got)
	}
	text, path, err := service.ReadFullLog(job.ID, 0)
	if err != nil {
		t.Fatalf("ReadFullLog returned error: %v", err)
	}
	if path != got.LogPath {
		t.Fatalf("expected path %q, got %q", got.LogPath, path)
	}
	if !strings.Contains(text, "1970-01-01T00:00:00Z") || !strings.Contains(text, "1970-01-01T00:01:07Z") {
		t.Fatalf("expected full log to retain oldest and newest lines:\n%s", text)
	}
	if _, err := os.Stat(filepath.Join(root, ".nexusdesk", "jobs", job.ID, "job.log")); err != nil {
		t.Fatalf("expected durable job log file: %v", err)
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

func TestServiceSurfacesJobPersistenceFailures(t *testing.T) {
	repo := &fakeJobRepository{saveErr: errors.New("disk full")}
	service := NewWithRepository(repo)
	job, _ := service.Start("task", "go test")

	issue, ok := service.PersistenceIssue()
	if !ok {
		t.Fatal("expected persistence issue")
	}
	if issue.JobID != job.ID || issue.Operation != "save job" || issue.Error != "disk full" || issue.At.IsZero() {
		t.Fatalf("unexpected persistence issue: %#v", issue)
	}
	if got, ok := service.Get(job.ID); !ok || got.Status != StatusRunning {
		t.Fatalf("job should remain available in memory after persistence failure: %#v", got)
	}

	repo.saveErr = nil
	service.AppendLog(job.ID, "retry save")
	if issue, ok := service.PersistenceIssue(); ok {
		t.Fatalf("expected successful save to clear persistence issue, got %#v", issue)
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

func TestServicePrunesTerminalJobsWithRetentionPolicy(t *testing.T) {
	now := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
	repo := &fakeJobRepository{}
	service := NewWithRepository(repo)
	for _, job := range []Job{
		{ID: "job-running", Kind: "task", Label: "running", Status: StatusRunning, StartedAt: now.Add(-2 * time.Hour)},
		{ID: "job-failed", Kind: "task", Label: "failed", Status: StatusFailed, StartedAt: now.Add(-4 * time.Hour), CompletedAt: now.Add(-3 * time.Hour)},
		{ID: "job-new", Kind: "task", Label: "new", Status: StatusSuccess, StartedAt: now.Add(-2 * time.Hour), CompletedAt: now.Add(-90 * time.Minute)},
		{ID: "job-old", Kind: "task", Label: "old", Status: StatusSuccess, StartedAt: now.Add(-72 * time.Hour), CompletedAt: now.Add(-71 * time.Hour)},
		{ID: "job-canceled", Kind: "task", Label: "canceled", Status: StatusCanceled, StartedAt: now.Add(-96 * time.Hour), CompletedAt: now.Add(-95 * time.Hour)},
	} {
		service.jobs = append(service.jobs, job)
	}

	result, err := service.Prune(RetentionPolicy{KeepRecent: 1, MaxAge: 48 * time.Hour, Now: now})
	if err != nil {
		t.Fatalf("Prune returned error: %v", err)
	}
	if result.Removed != 2 || len(repo.deleted) != 2 {
		t.Fatalf("expected two deleted jobs, result=%#v deleted=%#v", result, repo.deleted)
	}
	for _, id := range []string{"job-old", "job-canceled"} {
		if _, ok := service.Get(id); ok {
			t.Fatalf("expected %s to be pruned", id)
		}
	}
	for _, id := range []string{"job-running", "job-failed", "job-new"} {
		if _, ok := service.Get(id); !ok {
			t.Fatalf("expected %s to be retained", id)
		}
	}
}

func TestServicePruneCanIncludeFailuresWhenRequested(t *testing.T) {
	now := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
	service := New()
	service.jobs = []Job{
		{ID: "job-failed", Kind: "task", Label: "failed", Status: StatusFailed, CompletedAt: now.Add(-72 * time.Hour)},
	}
	result, err := service.Prune(RetentionPolicy{KeepRecent: 0, MaxAge: 24 * time.Hour, IncludeFailures: true, Now: now})
	if err != nil {
		t.Fatalf("Prune returned error: %v", err)
	}
	if result.Removed != 1 {
		t.Fatalf("expected failed job to be pruned when requested, got %#v", result)
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

func TestExternalAgentRunWorkflowIsAuditedAndExplicit(t *testing.T) {
	spec, ok := SlowWorkflowSpec(KindExternalAgentRun)
	if !ok {
		t.Fatal("expected external agent run workflow spec")
	}
	if !spec.RequiresDurableJob || !spec.RequiresExplicitStart || !spec.ProhibitedOnWorkspaceOpen || !spec.AuditRequired || !spec.Cancellable {
		t.Fatalf("external agent workflow is missing guardrails: %#v", spec)
	}
	if err := ValidateWorkflowStart(KindExternalAgentRun, StartOptions{}); err == nil {
		t.Fatal("expected implicit external agent workflow start to be rejected")
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
	listed  []Job
	saved   []Job
	deleted []string
	saveErr error
}

func (r *fakeJobRepository) SaveJob(job Job) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = append(r.saved, job)
	return nil
}

func (r *fakeJobRepository) ListJobs() ([]Job, error) {
	return append([]Job{}, r.listed...), nil
}

func (r *fakeJobRepository) DeleteJobs(ids []string) error {
	r.deleted = append(r.deleted, ids...)
	return nil
}
