package shell

import (
	"testing"

	"fyne.io/fyne/v2"
)

func TestNewToolPanelWidgetsInitializesSharedPanelState(t *testing.T) {
	widgets := newToolPanelWidgets()

	if widgets.problemResults == nil || widgets.problemStatus == nil {
		t.Fatalf("expected problem panel widgets to be initialized")
	}
	if widgets.taskResults == nil || widgets.taskStatus == nil || widgets.taskOutput == nil {
		t.Fatalf("expected task panel widgets to be initialized")
	}
	if widgets.chatHistoryResults == nil || widgets.chatHistoryStatus == nil || widgets.chatHistoryDetail == nil {
		t.Fatalf("expected chat history panel widgets to be initialized")
	}
	if widgets.historyResults == nil || widgets.historyStatus == nil || widgets.historyDetail == nil {
		t.Fatalf("expected history panel widgets to be initialized")
	}
	if widgets.agentAuditResults == nil || widgets.agentAuditStatus == nil || widgets.agentAuditDetail == nil {
		t.Fatalf("expected agent audit panel widgets to be initialized")
	}
	if widgets.approvalResults == nil || widgets.approvalStatus == nil || widgets.accessStatus == nil {
		t.Fatalf("expected approval panel widgets to be initialized")
	}
	if !widgets.taskOutput.Disabled() || widgets.taskOutput.Wrapping != fyne.TextWrapOff {
		t.Fatalf("expected task output to be read-only no-wrap monospace entry")
	}
	if !widgets.historyDetail.Disabled() || widgets.historyDetail.Wrapping != fyne.TextWrapWord {
		t.Fatalf("expected history detail to be read-only word-wrap monospace entry")
	}
}
