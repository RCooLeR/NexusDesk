package approvals

import (
	"errors"
	"testing"
	"time"
)

func TestAppendAndListApprovalRecords(t *testing.T) {
	root := t.TempDir()
	service := New()
	if _, err := service.Append(root, Record{Action: "write", Target: "notes.md"}); err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	records, err := service.List(root)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Risk != "medium" || records[0].Decision != "applied" || records[0].ID == "" {
		t.Fatalf("unexpected defaults: %#v", records[0])
	}
}

func TestAppendWritesRepositoryAndListPrefersRepository(t *testing.T) {
	root := t.TempDir()
	repository := &fakeApprovalRepository{}
	service := New()
	service.SetRepository(repository)
	if _, err := service.Append(root, Record{Action: "write", Target: "notes.md"}); err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	if len(repository.records) != 1 || repository.records[0].Action != "write" {
		t.Fatalf("expected repository save: %#v", repository.records)
	}
	repository.records = append([]Record{{
		ID:        "repo-only",
		Action:    "repo",
		Target:    "metadata",
		Risk:      "low",
		Decision:  "recorded",
		CreatedAt: time.Now().UTC(),
	}}, repository.records...)
	records, err := service.List(root)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if records[0].ID != "repo-only" {
		t.Fatalf("expected repository-backed list, got %#v", records)
	}
}

func TestAppendPersistsOnlyNewRepositoryRecord(t *testing.T) {
	root := t.TempDir()
	repository := &fakeApprovalRepository{}
	service := New()
	service.SetRepository(repository)

	if _, err := service.Append(root, Record{Action: "first", Target: "one.md"}); err != nil {
		t.Fatalf("first Append returned error: %v", err)
	}
	if _, err := service.Append(root, Record{Action: "second", Target: "two.md"}); err != nil {
		t.Fatalf("second Append returned error: %v", err)
	}

	if len(repository.records) != 2 {
		t.Fatalf("expected one repository save per append, got %#v", repository.records)
	}
	if repository.records[0].Action != "second" || repository.records[1].Action != "first" {
		t.Fatalf("unexpected repository record order: %#v", repository.records)
	}
}

func TestAppendSurfacesRepositorySaveFailure(t *testing.T) {
	root := t.TempDir()
	repository := &fakeApprovalRepository{saveErr: errors.New("metadata unavailable")}
	service := New()
	service.SetRepository(repository)

	if _, err := service.Append(root, Record{Action: "write", Target: "notes.md"}); err == nil {
		t.Fatal("expected repository save failure")
	}
}

type fakeApprovalRepository struct {
	records []Record
	saveErr error
}

func (r *fakeApprovalRepository) SaveApprovalRecord(record Record) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.records = append([]Record{record}, r.records...)
	return nil
}

func (r *fakeApprovalRepository) ListApprovalRecords(limit int) ([]Record, error) {
	if limit > 0 && len(r.records) > limit {
		return r.records[:limit], nil
	}
	return r.records, nil
}

func TestGrantAndRevokeFullProjectAccess(t *testing.T) {
	root := t.TempDir()
	service := New()
	policy, err := service.GrantFullProjectAccess(root, time.Hour)
	if err != nil {
		t.Fatalf("GrantFullProjectAccess returned error: %v", err)
	}
	if !policy.Active(time.Now().UTC()) || !service.HasFullProjectAccess(root) {
		t.Fatalf("expected active policy: %#v", policy)
	}
	policy, err = service.RevokeFullProjectAccess(root)
	if err != nil {
		t.Fatalf("RevokeFullProjectAccess returned error: %v", err)
	}
	if policy.Active(time.Now().UTC()) || service.HasFullProjectAccess(root) {
		t.Fatalf("expected inactive policy: %#v", policy)
	}
	records, err := service.List(root)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(records) != 2 || records[0].Decision != "revoked" || records[1].Decision != "granted" {
		t.Fatalf("unexpected access audit records: %#v", records)
	}
}

func TestLoadPolicyMarksExpiredAccessInactive(t *testing.T) {
	root := t.TempDir()
	service := New()
	expired := Policy{
		WorkspaceRoot:     root,
		FullProjectAccess: true,
		GrantedAt:         time.Now().UTC().Add(-2 * time.Hour),
		ExpiresAt:         time.Now().UTC().Add(-time.Hour),
	}
	if err := writeJSON(policyPath(root), expired); err != nil {
		t.Fatal(err)
	}
	policy, err := service.LoadPolicy(root)
	if err != nil {
		t.Fatalf("LoadPolicy returned error: %v", err)
	}
	if policy.FullProjectAccess || policy.Message == "" {
		t.Fatalf("expected expired policy to be inactive: %#v", policy)
	}
}
