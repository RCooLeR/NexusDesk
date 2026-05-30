package shell

import (
	"strings"
	"testing"
	"time"

	approvalsSvc "nexusdesk/internal/services/approvals"
)

func TestPolicyStatusText(t *testing.T) {
	active := policyStatusText(approvalsSvc.Policy{
		FullProjectAccess: true,
		GrantedAt:         time.Now().UTC(),
		ExpiresAt:         time.Now().UTC().Add(time.Hour),
	})
	if !strings.Contains(active, "active until") {
		t.Fatalf("expected active status, got %q", active)
	}
	inactive := policyStatusText(approvalsSvc.Policy{Message: "Full project access revoked."})
	if !strings.Contains(inactive, "inactive") || !strings.Contains(inactive, "revoked") {
		t.Fatalf("expected inactive status, got %q", inactive)
	}
}

func TestApprovalRowsEmpty(t *testing.T) {
	rows := approvalRows(nil)
	if len(rows) != 1 {
		t.Fatalf("expected one empty row, got %d", len(rows))
	}
}

func TestApprovalRecordCardDetails(t *testing.T) {
	record := approvalsSvc.Record{
		Action:    "agent-tool:write_file",
		Target:    "docs/report.md",
		Risk:      "high",
		Decision:  "approved",
		Message:   "Per-call agent tool approval",
		CreatedAt: time.Date(2026, 5, 30, 12, 34, 0, 0, time.UTC),
	}

	subtitle := approvalRecordSubtitle(record)
	for _, expected := range []string{"target: docs/report.md", "decision: approved", "2026-05-30 12:34:00"} {
		if !strings.Contains(subtitle, expected) {
			t.Fatalf("expected approval card subtitle to contain %q, got %q", expected, subtitle)
		}
	}
	details := approvalRecordDetails(record)
	for _, expected := range []string{"Risk: high", "Details: Per-call agent tool approval"} {
		if !strings.Contains(details, expected) {
			t.Fatalf("expected approval card details to contain %q, got %q", expected, details)
		}
	}
}
