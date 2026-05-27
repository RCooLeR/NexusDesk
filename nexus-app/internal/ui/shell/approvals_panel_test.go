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
