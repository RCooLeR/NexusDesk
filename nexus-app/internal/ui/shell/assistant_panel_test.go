package shell

import (
	"strings"
	"testing"
	"time"

	"nexusdesk/internal/domain"
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
	if !strings.Contains(text, "Answer") || !strings.Contains(text, "No explicit source context") || !strings.Contains(text, "Evidence: weak") {
		t.Fatalf("expected weak-evidence warning, got %q", text)
	}
	withSources := assistantResponseMarkdown(assistantSvc.Result{Message: "Answer", Model: "qwen", ContextRelPath: "context: README.md", SourcePaths: []string{"README.md"}})
	if strings.Contains(withSources, "No explicit source context") {
		t.Fatalf("did not expect weak-evidence warning with sources, got %q", withSources)
	}
	for _, expected := range []string{"Model: `qwen`", "Context: `context: README.md`", "Sources: `README.md`", "Evidence: source-backed (1 source(s), no line citations detected)."} {
		if !strings.Contains(withSources, expected) {
			t.Fatalf("expected source/model footer to contain %q, got %q", expected, withSources)
		}
	}
}

func TestAssistantResponseMarkdownIncludesLineCitations(t *testing.T) {
	text := assistantResponseMarkdown(assistantSvc.Result{
		Message:        "Use README.md:12 and docs/guide.md#L4-L6. Ignore other.md:1.",
		Model:          "qwen",
		ContextRelPath: "pack: README.md, docs/guide.md",
	})

	for _, expected := range []string{"Citations: `README.md:L12`, `docs/guide.md:L4-L6`", "Unverified citations: `other.md:L1`", "Sources: `README.md`, `docs/guide.md`", "Evidence: line-cited (2 source(s), 2 line ref(s); 1 citation outside selected sources)."} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected citation footer to contain %q, got %q", expected, text)
		}
	}
	refs := assistantCitationRefs(assistantSvc.Result{
		Message:        "Use README.md:12 and docs/guide.md#L4-L6. Ignore other.md:1.",
		ContextRelPath: "pack: README.md, docs/guide.md",
	})
	if strings.Join(refs, "|") != "README.md:L12|docs/guide.md:L4-L6" {
		t.Fatalf("unexpected citation refs: %#v", refs)
	}
	unverified := assistantUnverifiedCitationRefs(assistantSvc.Result{
		Message:        "Use README.md:12 and docs/guide.md#L4-L6. Ignore other.md:1.",
		ContextRelPath: "pack: README.md, docs/guide.md",
	})
	if strings.Join(unverified, "|") != "other.md:L1" {
		t.Fatalf("unexpected unverified citation refs: %#v", unverified)
	}
}

func TestAssistantEvidenceDiagnosticClassifiesSourceQuality(t *testing.T) {
	cases := []struct {
		name    string
		result  assistantSvc.Result
		quality string
		summary string
		sources int
		refs    int
	}{
		{
			name:    "weak",
			result:  assistantSvc.Result{Message: "Answer"},
			quality: "weak",
			summary: "no explicit source context",
		},
		{
			name:    "source-backed",
			result:  assistantSvc.Result{Message: "Answer", SourcePaths: []string{"README.md"}},
			quality: "source-backed",
			summary: "no line citations detected",
			sources: 1,
		},
		{
			name:    "line-cited",
			result:  assistantSvc.Result{Message: "See README.md:12.", SourcePaths: []string{"README.md"}},
			quality: "line-cited",
			summary: "1 line ref",
			sources: 1,
			refs:    1,
		},
		{
			name:    "unverified",
			result:  assistantSvc.Result{Message: "See missing.md:12.", SourcePaths: []string{"README.md"}},
			quality: "source-backed",
			summary: "no verified line citations",
			sources: 1,
			refs:    0,
		},
	}
	for _, tt := range cases {
		got := assistantEvidenceDiagnosticForResult(tt.result)
		if got.Quality != tt.quality || !strings.Contains(got.Summary, tt.summary) || got.SourceCount != tt.sources || got.CitationCount != tt.refs {
			t.Fatalf("%s diagnostic = %#v", tt.name, got)
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

func TestAssistantCitationRefsDedupesAndNormalizes(t *testing.T) {
	refs := assistantCitationRefs(assistantSvc.Result{
		Message:     "See docs\\guide.md:10, docs/guide.md#L10, and docs/guide.md#L11-L12.",
		SourcePaths: []string{"docs"},
	})
	want := []string{"docs/guide.md:L10", "docs/guide.md:L11-L12"}
	if strings.Join(refs, "|") != strings.Join(want, "|") {
		t.Fatalf("unexpected citation refs: %#v", refs)
	}
}

func TestAssistantCitationRefsWithoutSourcesAreUnverified(t *testing.T) {
	result := assistantSvc.Result{Message: "See README.md#L7."}
	if refs := assistantCitationRefs(result); len(refs) != 0 {
		t.Fatalf("expected no verified refs without sources, got %#v", refs)
	}
	unverified := assistantUnverifiedCitationRefs(result)
	if strings.Join(unverified, "|") != "README.md:L7" {
		t.Fatalf("unexpected unverified refs: %#v", unverified)
	}
	diagnostic := assistantEvidenceDiagnosticForResult(result)
	if diagnostic.Quality != "weak" || diagnostic.UnverifiedCitationCount != 1 || !strings.Contains(diagnostic.Summary, "1 unverified line ref") {
		t.Fatalf("unexpected diagnostic: %#v", diagnostic)
	}
}

func TestAssistantCitationSnippetsExtractsBoundedSourceLines(t *testing.T) {
	previewer := assistantCitationFakePreviewer{files: map[string]string{
		"README.md":     "one\ntwo\nthree\nfour\nfive\nsix\n",
		"docs/guide.md": "alpha\nbeta\n",
	}}
	result := assistantSvc.Result{
		Message:     "See README.md#L2-L5 and docs/guide.md:2.",
		SourcePaths: []string{"README.md", "docs/guide.md"},
	}
	snippets := assistantCitationSnippets("workspace", result, previewer)
	if len(snippets) != 2 {
		t.Fatalf("expected two snippets, got %#v", snippets)
	}
	for _, expected := range []string{"README.md:L2-L5", "L2: two", "L5: five", "docs/guide.md:L2", "L2: beta"} {
		if !strings.Contains(strings.Join(snippets, "\n"), expected) {
			t.Fatalf("expected snippets to contain %q, got %#v", expected, snippets)
		}
	}
}

func TestAssistantCitationSnippetsSkipsMissingSources(t *testing.T) {
	previewer := assistantCitationFakePreviewer{files: map[string]string{"README.md": "one\n"}}
	result := assistantSvc.Result{
		Message:     "See README.md:4 and missing.md:1.",
		SourcePaths: []string{"README.md", "missing.md"},
	}
	if snippets := assistantCitationSnippets("workspace", result, previewer); len(snippets) != 0 {
		t.Fatalf("expected missing/out-of-range snippets to be skipped, got %#v", snippets)
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

type assistantCitationFakePreviewer struct {
	files map[string]string
}

func (p assistantCitationFakePreviewer) PreviewFile(root string, relPath string) (domain.FilePreview, error) {
	text, ok := p.files[relPath]
	if !ok {
		return domain.FilePreview{}, errAssistantCitationFakeMissing{}
	}
	return domain.FilePreview{RelPath: relPath, Text: text}, nil
}

type errAssistantCitationFakeMissing struct{}

func (errAssistantCitationFakeMissing) Error() string { return "missing" }
