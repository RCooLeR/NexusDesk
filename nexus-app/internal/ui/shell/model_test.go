package shell

import (
	"testing"

	"nexusdesk/internal/domain"
	"nexusdesk/internal/services/llm"
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

func TestStateAssistantConversationResetsAndCaps(t *testing.T) {
	state := NewState()
	state.SetAssistantConversation([]llm.ChatTurn{{Role: "user", Content: "one"}})
	state.AppendAssistantExchange("two", "three")
	if got := state.AssistantConversation(); len(got) != 3 {
		t.Fatalf("unexpected conversation after append: %#v", got)
	}
	for index := range 20 {
		state.AppendAssistantExchange("prompt", string(rune('a'+index)))
	}
	if got := state.AssistantConversation(); len(got) != 24 {
		t.Fatalf("conversation should be capped to 24 turns, got %d", len(got))
	}
	state.SetWorkspace(domain.Workspace{Root: "C:/repo", Name: "repo"})
	if got := state.AssistantConversation(); len(got) != 0 {
		t.Fatalf("workspace change should reset conversation, got %#v", got)
	}
}
