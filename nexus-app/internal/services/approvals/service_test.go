package approvals

import (
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
