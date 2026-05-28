package shell

import (
	"strings"
	"testing"
	"time"

	agentSvc "nexusdesk/internal/services/agent"
	assistantSvc "nexusdesk/internal/services/assistant"
	llmSvc "nexusdesk/internal/services/llm"
	metadataSvc "nexusdesk/internal/services/metadata"
	settingsSvc "nexusdesk/internal/services/settings"
)

func TestAgentActivityTailKeepsLastTwoMessages(t *testing.T) {
	tail := agentActivityTail{}
	tail.Add("one")
	tail.Add("two")
	tail.Add("three")
	text := tail.Markdown()
	if strings.Contains(text, "one") || !strings.Contains(text, "two") || !strings.Contains(text, "three") {
		t.Fatalf("unexpected tail: %q", text)
	}
}

func TestAgentEventLineFormatsUsefulEvents(t *testing.T) {
	cases := []struct {
		event agentSvc.Event
		want  string
	}{
		{event: agentSvc.Event{Type: "model_request", Iteration: 2}, want: "Thinking, step 2"},
		{event: agentSvc.Event{Type: "tool_start", ToolName: "read_context"}, want: "Tool requested: read_context"},
		{event: agentSvc.Event{Type: "plan_update"}, want: "Plan updated."},
	}
	for _, tt := range cases {
		got := agentEventLine(tt.event)
		if !strings.Contains(got, tt.want) {
			t.Fatalf("agentEventLine(%#v) = %q, want %q", tt.event, got, tt.want)
		}
	}
}

func TestAgentEventLineGuardsEmptyToolNames(t *testing.T) {
	for _, eventType := range []string{"tool_start", "tool_done", "tool_error"} {
		line := agentEventLine(agentSvc.Event{Type: eventType})
		if strings.HasSuffix(line, ": ") || !strings.Contains(line, "unknown tool") {
			t.Fatalf("expected guarded empty tool name for %s, got %q", eventType, line)
		}
	}
}

func TestAgentToolApprovalMessageSummarizesRiskAndTarget(t *testing.T) {
	message := agentToolApprovalMessage(agentSvc.ToolApprovalRequest{
		Name:        "write_file",
		Risk:        "high",
		Description: "Write a file",
		Args:        map[string]string{"relPath": "docs/report.md"},
	})
	for _, expected := range []string{"write_file", "high", "Write a file", "docs/report.md", "single tool call"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected approval message to contain %q, got %q", expected, message)
		}
	}
}

func TestAgentFinalMarkdownIncludesStopReason(t *testing.T) {
	text := agentFinalMarkdown(agentSvc.Result{Message: "Done", StopReason: "safety_guard"})
	if !strings.Contains(text, "Done") || !strings.Contains(text, "safety_guard") {
		t.Fatalf("unexpected final markdown: %q", text)
	}
}

func TestChatTurnsFromMetadataKeepsValidConversationTurns(t *testing.T) {
	turns := chatTurnsFromMetadata([]metadataSvc.ChatMessageRecord{
		{Role: "user", Content: " Hello ", CreatedAt: time.Now()},
		{Role: "system", Content: "ignored"},
		{Role: "assistant", Content: "World"},
	})
	if len(turns) != 2 || turns[0].Role != "user" || turns[1].Content != "World" {
		t.Fatalf("unexpected turns: %#v", turns)
	}
}

func TestChatTurnPreviewCompactsLongContent(t *testing.T) {
	preview := chatTurnPreview(llmSvc.ChatTurn{Role: "assistant", Content: strings.Repeat("word ", 40)})
	if !strings.HasPrefix(preview, "Assistant: ") || len(preview) > 105 {
		t.Fatalf("unexpected preview: %q", preview)
	}
}

func TestChatTurnPreviewGuardsEmptyRoleAndContent(t *testing.T) {
	preview := chatTurnPreview(llmSvc.ChatTurn{})
	if preview != "Turn: (empty)" {
		t.Fatalf("unexpected empty turn preview: %q", preview)
	}
}

func TestAssistantResponseMarkdownWarnsWithoutSources(t *testing.T) {
	text := assistantResponseMarkdown(assistantSvc.Result{Message: "Answer"})
	if !strings.Contains(text, "Answer") || !strings.Contains(text, "No explicit source context") {
		t.Fatalf("expected weak-evidence warning, got %q", text)
	}
	withSources := assistantResponseMarkdown(assistantSvc.Result{Message: "Answer", Model: "qwen", ContextRelPath: "context: README.md", SourcePaths: []string{"README.md"}})
	if strings.Contains(withSources, "No explicit source context") {
		t.Fatalf("did not expect weak-evidence warning with sources, got %q", withSources)
	}
	for _, expected := range []string{"Model: `qwen`", "Context: `context: README.md`", "Sources: `README.md`"} {
		if !strings.Contains(withSources, expected) {
			t.Fatalf("expected source/model footer to contain %q, got %q", expected, withSources)
		}
	}
}

func TestAssistantSourcePathsFromContextPortsWailsRules(t *testing.T) {
	tests := []struct {
		context string
		want    []string
	}{
		{context: "pack: README.md, docs/guide.md", want: []string{"README.md", "docs/guide.md"}},
		{context: "dir: docs (3 files)", want: []string{"docs"}},
		{context: "context: README.md", want: []string{"README.md"}},
		{context: "project: .", want: []string{"."}},
		{context: "context: 2 roots", want: nil},
		{context: "agent", want: nil},
	}
	for _, tt := range tests {
		got := assistantSourcePathsFromContext(tt.context)
		if strings.Join(got, "|") != strings.Join(tt.want, "|") {
			t.Fatalf("assistantSourcePathsFromContext(%q) = %#v, want %#v", tt.context, got, tt.want)
		}
	}
}

func TestAssistantEffectiveSourcePathsFallsBackAndDedupes(t *testing.T) {
	paths := assistantEffectiveSourcePaths(assistantSvc.Result{
		ContextRelPath: "pack: README.md, docs/guide.md",
		SourcePaths:    []string{"README.md", " README.md ", "agent", "docs/guide.md"},
	})
	if len(paths) != 2 || paths[0] != "README.md" || paths[1] != "docs/guide.md" {
		t.Fatalf("unexpected explicit source paths: %#v", paths)
	}
	fallback := assistantEffectiveSourcePaths(assistantSvc.Result{ContextRelPath: "pack: README.md, docs/guide.md"})
	if len(fallback) != 2 || fallback[0] != "README.md" || fallback[1] != "docs/guide.md" {
		t.Fatalf("unexpected fallback source paths: %#v", fallback)
	}
}

func TestCompareLatestAssistantPromptCarriesPromptAndAnswer(t *testing.T) {
	text := compareLatestAssistantPrompt("What changed?", "A changed.")
	for _, expected := range []string{"Compare the previous assistant answer", "Original prompt:", "What changed?", "Previous assistant answer:", "A changed.", "recommended final answer"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected compare prompt to contain %q, got %q", expected, text)
		}
	}
}

func TestAssistantProfileOptionRoundTripsID(t *testing.T) {
	profile := assistantSvc.DefaultProfile()
	option := assistantProfileOption(profile.PromptProfiles[1])
	if option != "Reviewer" {
		t.Fatalf("unexpected option label: %q", option)
	}
	if got := assistantProfileIDFromOption(option, profile); got != "reviewer" {
		t.Fatalf("unexpected option id: %q", got)
	}
}

func TestAssistantContextPathsForRequestPrefersPins(t *testing.T) {
	paths := assistantContextPathsForRequest([]string{" README.md ", "docs", "README.md"}, "selected.go")
	if len(paths) != 2 || paths[0] != "README.md" || paths[1] != "docs" {
		t.Fatalf("unexpected pinned paths: %#v", paths)
	}
}

func TestAssistantContextPathsForRequestFallsBackToSelected(t *testing.T) {
	paths := assistantContextPathsForRequest(nil, "selected.go")
	if len(paths) != 1 || paths[0] != "selected.go" {
		t.Fatalf("unexpected selected fallback: %#v", paths)
	}
}

func TestAgentContextBudgetBytesUsesModelBudget(t *testing.T) {
	store := shellSettingsStore{settings: settingsSvc.Settings{ContextTokens: 1000, ResponseReserveTokens: 250}}
	if got := agentContextBudgetBytes(store); got != 3000 {
		t.Fatalf("unexpected budget bytes: %d", got)
	}
}

type shellSettingsStore struct {
	settings settingsSvc.Settings
}

func (s shellSettingsStore) Load() (settingsSvc.Settings, error) {
	return s.settings, nil
}
