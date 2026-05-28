package shell

import (
	"testing"

	jobsSvc "nexusdesk/internal/services/jobs"
)

func TestWorkspaceOpenPolicyAllowsOnlySafeActions(t *testing.T) {
	allowed := []workspaceOpenActionKind{
		workspaceOpenActionJobsRefresh,
		workspaceOpenActionChatHistoryRefresh,
		workspaceOpenActionAgentAuditRefresh,
		workspaceOpenActionUnifiedHistoryRefresh,
		workspaceOpenActionApprovalsRefresh,
		workspaceOpenActionNavigatorRefresh,
		workspaceOpenActionAssistantPinsRefresh,
		workspaceOpenActionCompatibilityImport,
	}
	for _, action := range allowed {
		if !isWorkspaceOpenActionAllowed(action) {
			t.Fatalf("expected workspace-open action %q to be allowed", action)
		}
	}
}

func TestWorkspaceOpenPolicyRejectsHeavyActions(t *testing.T) {
	disallowed := []workspaceOpenActionKind{
		"git-refresh",
		"docker-inspect",
		"connector-query",
		"assistant-model-call",
		"shell-command",
		"dump-import",
		"ocr-extract",
		"deep-indexing",
	}
	for _, action := range disallowed {
		if isWorkspaceOpenActionAllowed(action) {
			t.Fatalf("expected heavy workspace-open action %q to be blocked", action)
		}
	}
}

func TestWorkspaceOpenPolicyRejectsSlowWorkflowJobKinds(t *testing.T) {
	for _, spec := range jobsSvc.SlowWorkflowSpecs() {
		if isWorkspaceOpenActionAllowed(workspaceOpenActionKind(spec.Kind)) {
			t.Fatalf("expected slow workflow %q to be blocked on workspace open", spec.Kind)
		}
	}
}
