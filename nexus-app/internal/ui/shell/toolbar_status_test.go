package shell

import (
	"strings"
	"testing"

	"nexusdesk/internal/domain"
	gitSvc "nexusdesk/internal/services/git"
	settingsSvc "nexusdesk/internal/services/settings"
)

func TestToolbarStatusTextSummarizesChromeState(t *testing.T) {
	snapshot := toolbarStatusSnapshot{
		Workspace: domain.Workspace{Name: "NexusDesk"},
		GitStatus: gitSvc.Status{
			Available:   true,
			Branch:      "main",
			Head:        "1234567890abcdef",
			AheadBehind: "ahead 1",
		},
		Settings: settingsSvc.Settings{
			Provider: "ollama",
			Model:    "qwen3-coder:30b",
		},
	}

	if got := toolbarWorkspaceText(snapshot); got != "Workspace: NexusDesk" {
		t.Fatalf("unexpected workspace text: %q", got)
	}
	if got := toolbarBranchText(snapshot); got != "Branch: main @ 1234567 ahead 1" {
		t.Fatalf("unexpected branch text: %q", got)
	}
	if got := toolbarProviderText(snapshot); got != "Model: ollama/qwen3-coder:30b" {
		t.Fatalf("unexpected provider text: %q", got)
	}
}

func TestToolbarStatusTextFallsBackForColdStart(t *testing.T) {
	snapshot := toolbarStatusSnapshot{}
	for _, got := range []string{
		toolbarWorkspaceText(snapshot),
		toolbarBranchText(snapshot),
		toolbarProviderText(snapshot),
	} {
		if strings.TrimSpace(got) == "" {
			t.Fatalf("expected non-empty toolbar fallback")
		}
	}
	if got := toolbarWorkspaceText(snapshot); got != "Workspace: none" {
		t.Fatalf("unexpected workspace fallback: %q", got)
	}
	if got := toolbarBranchText(snapshot); got != "Branch: refresh Git" {
		t.Fatalf("unexpected branch fallback: %q", got)
	}
	if got := toolbarProviderText(toolbarStatusSnapshot{SettingsError: "bad settings"}); got != "Model: settings error" {
		t.Fatalf("unexpected provider error fallback: %q", got)
	}
}
