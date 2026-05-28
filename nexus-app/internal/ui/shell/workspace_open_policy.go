package shell

import jobsSvc "nexusdesk/internal/services/jobs"

type workspaceOpenActionKind string

const (
	workspaceOpenActionJobsRefresh           workspaceOpenActionKind = "jobs-refresh"
	workspaceOpenActionChatHistoryRefresh    workspaceOpenActionKind = "chat-history-refresh"
	workspaceOpenActionAgentAuditRefresh     workspaceOpenActionKind = "agent-audit-refresh"
	workspaceOpenActionUnifiedHistoryRefresh workspaceOpenActionKind = "unified-history-refresh"
	workspaceOpenActionApprovalsRefresh      workspaceOpenActionKind = "approvals-refresh"
	workspaceOpenActionNavigatorRefresh      workspaceOpenActionKind = "navigator-refresh"
	workspaceOpenActionAssistantPinsRefresh  workspaceOpenActionKind = "assistant-pins-refresh"
	workspaceOpenActionCompatibilityImport   workspaceOpenActionKind = "metadata-compat-import"
)

func isWorkspaceOpenActionAllowed(kind workspaceOpenActionKind) bool {
	if jobsSvc.ProhibitedOnWorkspaceOpen(string(kind)) {
		return false
	}
	switch kind {
	case workspaceOpenActionJobsRefresh,
		workspaceOpenActionChatHistoryRefresh,
		workspaceOpenActionAgentAuditRefresh,
		workspaceOpenActionUnifiedHistoryRefresh,
		workspaceOpenActionApprovalsRefresh,
		workspaceOpenActionNavigatorRefresh,
		workspaceOpenActionAssistantPinsRefresh,
		workspaceOpenActionCompatibilityImport:
		return true
	default:
		return false
	}
}

func (v *View) runWorkspaceOpenAction(kind workspaceOpenActionKind, action func()) bool {
	if !isWorkspaceOpenActionAllowed(kind) || action == nil {
		return false
	}
	action()
	return true
}
