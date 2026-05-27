package shell

import (
	"testing"

	"nexusdesk/internal/domain"
)

func TestStateAssistantContextPathsDeduplicateAndResetWithWorkspace(t *testing.T) {
	state := NewState()
	if !state.AddAssistantContextPath("README.md") {
		t.Fatal("expected first context path to be added")
	}
	if state.AddAssistantContextPath("README.md") {
		t.Fatal("expected duplicate context path to be ignored")
	}
	if state.AddAssistantContextPath("   ") {
		t.Fatal("expected empty context path to be ignored")
	}
	paths := state.AssistantContextPaths()
	if len(paths) != 1 || paths[0] != "README.md" {
		t.Fatalf("unexpected context paths: %#v", paths)
	}

	state.SetWorkspace(domain.Workspace{Root: "C:/repo", Name: "repo"})
	if got := state.AssistantContextPaths(); len(got) != 0 {
		t.Fatalf("workspace change should reset context pins, got %#v", got)
	}
}

func TestStateRemoveAssistantContextPath(t *testing.T) {
	state := NewState()
	state.AddAssistantContextPath("README.md")
	state.AddAssistantContextPath("docs")
	if !state.RemoveAssistantContextPath("README.md") {
		t.Fatal("expected path removal")
	}
	if state.RemoveAssistantContextPath("missing.md") {
		t.Fatal("missing path should not report removal")
	}
	paths := state.AssistantContextPaths()
	if len(paths) != 1 || paths[0] != "docs" {
		t.Fatalf("unexpected context paths after remove: %#v", paths)
	}
}
