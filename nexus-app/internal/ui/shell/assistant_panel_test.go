package shell

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	agentSvc "nexusdesk/internal/services/agent"
	assistantSvc "nexusdesk/internal/services/assistant"
	llmSvc "nexusdesk/internal/services/llm"
	metadataSvc "nexusdesk/internal/services/metadata"
	settingsSvc "nexusdesk/internal/services/settings"
)

func TestAssistantControllerInitialState(t *testing.T) {
	view := &View{}
	controller := newAssistantController(view)

	if controller.view != view {
		t.Fatal("expected controller to retain parent view")
	}
	if controller.prompt != nil || controller.runStatus != nil || controller.modelRoute != nil {
		t.Fatalf("expected lazy widget state before panel construction, got %#v", controller)
	}
}

func TestAssistantHeaderKeepsModeRouteStatusVisible(t *testing.T) {
	status := widget.NewLabel("Ready: Ask. Model route: Auto. Context: no explicit context.")
	header := newAssistantHeader(status)

	if !canvasObjectContains(header, status) {
		t.Fatal("expected assistant header to contain the live run status label")
	}
	if !canvasObjectContainsLabel(header, "Assistant") {
		t.Fatal("expected assistant header title")
	}
}

func TestAssistantPanelLayoutKeepsComposerPinnedToBottom(t *testing.T) {
	header := widget.NewLabel("header")
	composer := widget.NewLabel("composer")
	sidebar := widget.NewLabel("sidebar")
	messages := widget.NewLabel("messages")

	panel := newAssistantPanelLayout(header, composer, sidebar, messages)

	if len(panel.Objects) != 4 {
		t.Fatalf("expected four assistant layout objects, got %d", len(panel.Objects))
	}
	if panel.Objects[0] != messages || panel.Objects[1] != header || panel.Objects[2] != composer || panel.Objects[3] != sidebar {
		t.Fatalf("unexpected assistant layout object order: %#v", panel.Objects)
	}
}

func TestSelectedAssistantModelRouteOptionHandlesMissingController(t *testing.T) {
	if got := selectedAssistantModelRouteOption(&View{}); got != assistantAutoModelRouteLabel {
		t.Fatalf("expected auto route without assistant controller, got %q", got)
	}
	if got := (&View{}).selectedAssistantModelRouteID("write Go tests"); got != "" {
		t.Fatalf("expected no explicit route without assistant controller, got %q", got)
	}
}

func canvasObjectContains(root fyne.CanvasObject, target fyne.CanvasObject) bool {
	if root == target {
		return true
	}
	container, ok := root.(*fyne.Container)
	if !ok {
		return false
	}
	for _, child := range container.Objects {
		if canvasObjectContains(child, target) {
			return true
		}
	}
	return false
}

func canvasObjectContainsLabel(root fyne.CanvasObject, text string) bool {
	if label, ok := root.(*widget.Label); ok && label.Text == text {
		return true
	}
	container, ok := root.(*fyne.Container)
	if !ok {
		return false
	}
	for _, child := range container.Objects {
		if canvasObjectContainsLabel(child, text) {
			return true
		}
	}
	return false
}

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

func TestAssistantStreamRendererCoalescesDeltasUntilFlush(t *testing.T) {
	rendered := make([]string, 0, 2)
	stream := newAssistantStreamRendererWithRender(func(text string) {
		rendered = append(rendered, text)
	}, time.Hour)
	defer stream.Stop()

	stream.Append("hello")
	stream.Append(" world")
	if len(rendered) != 0 {
		t.Fatalf("expected no render before flush, got %#v", rendered)
	}
	stream.Flush()
	if len(rendered) != 1 || rendered[0] != "hello world" {
		t.Fatalf("expected coalesced render, got %#v", rendered)
	}
	stream.Flush()
	if len(rendered) != 1 {
		t.Fatalf("expected clean flush to skip duplicate render, got %#v", rendered)
	}
	stream.Append("!")
	stream.Flush()
	if len(rendered) != 2 || rendered[1] != "hello world!" {
		t.Fatalf("expected full stream text on next render, got %#v", rendered)
	}
}

func TestAssistantStreamRendererHandlesLongDeltaBurst(t *testing.T) {
	rendered := make([]string, 0, 1)
	stream := newAssistantStreamRendererWithRender(func(text string) {
		rendered = append(rendered, text)
	}, time.Hour)
	defer stream.Stop()

	var expected strings.Builder
	for index := 0; index < 2000; index++ {
		delta := "chunk-" + strconv.Itoa(index) + "\n"
		expected.WriteString(delta)
		stream.Append(delta)
	}
	stream.Flush()

	if len(rendered) != 1 {
		t.Fatalf("expected long stream burst to render once, got %d renders", len(rendered))
	}
	if rendered[0] != expected.String() {
		t.Fatalf("expected rendered stream length %d, got %d", expected.Len(), len(rendered[0]))
	}
}

func TestAgentEventRendererCoalescesLinesUntilFlush(t *testing.T) {
	type renderCall struct {
		markdown string
		lines    []string
	}
	calls := make([]renderCall, 0, 2)
	renderer := newAgentEventRenderer(func(markdown string, lines []string) {
		calls = append(calls, renderCall{markdown: markdown, lines: lines})
	}, time.Hour)
	defer renderer.Stop()

	renderer.Append("one")
	renderer.Append("two")
	renderer.Append("three")
	if len(calls) != 0 {
		t.Fatalf("expected no render before flush, got %#v", calls)
	}
	renderer.Flush()
	if len(calls) != 1 {
		t.Fatalf("expected one coalesced render, got %#v", calls)
	}
	if got := strings.Join(calls[0].lines, ","); got != "one,two,three" {
		t.Fatalf("expected all pending lines in first batch, got %q", got)
	}
	if strings.Contains(calls[0].markdown, "one") || !strings.Contains(calls[0].markdown, "two") || !strings.Contains(calls[0].markdown, "three") {
		t.Fatalf("expected markdown to show latest tail, got %q", calls[0].markdown)
	}
	renderer.Flush()
	if len(calls) != 1 {
		t.Fatalf("expected clean flush to skip duplicate render, got %#v", calls)
	}
	renderer.Append("four")
	renderer.Flush()
	if len(calls) != 2 || strings.Join(calls[1].lines, ",") != "four" {
		t.Fatalf("expected second batch to include new line only, got %#v", calls)
	}
	if !strings.Contains(calls[1].markdown, "three") || !strings.Contains(calls[1].markdown, "four") {
		t.Fatalf("expected markdown to keep latest event tail, got %q", calls[1].markdown)
	}
}

func TestAgentEventRendererBatchesBurst(t *testing.T) {
	batchCount := 0
	lineCount := 0
	lastMarkdown := ""
	renderer := newAgentEventRenderer(func(markdown string, lines []string) {
		batchCount++
		lineCount += len(lines)
		lastMarkdown = markdown
	}, time.Hour)
	defer renderer.Stop()

	for index := 1; index <= 500; index++ {
		renderer.Append("event-" + strconv.Itoa(index))
	}
	renderer.Flush()

	if batchCount != 1 {
		t.Fatalf("expected one render batch, got %d", batchCount)
	}
	if lineCount != 500 {
		t.Fatalf("expected all burst lines in batch, got %d", lineCount)
	}
	if strings.Contains(lastMarkdown, "event-498") || !strings.Contains(lastMarkdown, "event-499") || !strings.Contains(lastMarkdown, "event-500") {
		t.Fatalf("expected visible markdown to keep only newest event tail, got %q", lastMarkdown)
	}
}

func TestAgentEventLineFormatsUsefulEvents(t *testing.T) {
	cases := []struct {
		event agentSvc.Event
		want  string
	}{
		{event: agentSvc.Event{Type: "model_request", Iteration: 2}, want: "Timeline: step 2 - asking model"},
		{event: agentSvc.Event{Type: "tool_start", Iteration: 2, ToolName: "read_context"}, want: "Timeline: step 2 - tool requested: read_context."},
		{event: agentSvc.Event{Type: "tool_error", Iteration: 3, ToolName: "write_file", Error: "denied"}, want: "Timeline: step 3 - tool failed: write_file. denied"},
		{event: agentSvc.Event{Type: "plan_update", Iteration: 4}, want: "Timeline: step 4 - plan updated."},
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
	subtitle := agentToolApprovalSubtitle(agentSvc.ToolApprovalRequest{
		Name: "write_file",
		Risk: "high",
		Args: map[string]string{"relPath": "docs/report.md"},
	})
	for _, expected := range []string{"write_file", "risk: high", "target: docs/report.md"} {
		if !strings.Contains(subtitle, expected) {
			t.Fatalf("expected approval card subtitle to contain %q, got %q", expected, subtitle)
		}
	}
}

func TestAgentFinalMarkdownIncludesStopReason(t *testing.T) {
	text := agentFinalMarkdown(agentSvc.Result{Message: "Done", Model: "qwen3-coder:30b", ModelRoute: "Main coding model", StopReason: "safety_guard"})
	if !strings.Contains(text, "Done") || !strings.Contains(text, "qwen3-coder:30b") || !strings.Contains(text, "Main coding model") || !strings.Contains(text, "safety_guard") {
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
	withSources := assistantResponseMarkdown(assistantSvc.Result{Message: "Answer", Model: "qwen", ModelRoute: "Main coding model", ContextRelPath: "context: README.md", SourcePaths: []string{"README.md"}})
	if strings.Contains(withSources, "No explicit source context") {
		t.Fatalf("did not expect weak-evidence warning with sources, got %q", withSources)
	}
	for _, expected := range []string{"Model: `qwen`", "Model route: `Main coding model`", "Context: `context: README.md`", "Sources: `README.md`", "Evidence: source-backed (1 source(s), no line citations detected, cited 0/1 source(s))."} {
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

	for _, expected := range []string{"Citations: `README.md:L12`, `docs/guide.md:L4-L6`", "Unverified citations: `other.md:L1`", "Sources: `README.md`, `docs/guide.md`", "Evidence: line-cited (2 source(s), 2 line ref(s), cited 2/2 source(s); 1 citation outside selected sources)."} {
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

func TestAssistantPreRunStatusLineSummarizesModeRouteAndContext(t *testing.T) {
	store := shellSettingsStore{settings: settingsSvc.Settings{
		ContextTokens:         1000,
		ResponseReserveTokens: 250,
		ModelRoutes: []settingsSvc.ModelRoute{{
			ID:                    settingsSvc.RouteCSVExcelScripts,
			Label:                 "CSV / Excel data scripts",
			Model:                 "qwen3-coder:30b",
			ContextTokens:         4000,
			ResponseReserveTokens: 1000,
		}},
	}}

	line := assistantPreRunStatusLine(store, "Ask", assistantAutoModelRouteLabel, "", []string{"data/sales.csv"}, "")

	for _, expected := range []string{"Ready: Ask.", "Auto -> CSV / Excel data scripts", "Context: 1 pinned context root(s)."} {
		if !strings.Contains(line, expected) {
			t.Fatalf("expected %q in pre-run status %q", expected, line)
		}
	}
}

func TestAssistantResultStatusLineSummarizesEvidenceAndWarnings(t *testing.T) {
	line := assistantResultStatusLine(assistantSvc.Result{
		Message:      "See README.md:12.",
		Model:        "qwen",
		ModelRoute:   "Main coding model",
		SourcePaths:  []string{"README.md", "docs/guide.md"},
		RouteWarning: "route fallback used",
	})

	for _, expected := range []string{"Completed: qwen via Main coding model.", "Evidence: line-cited", "Sources: 2", "verified refs: 1", "unverified refs: 0", "Route warning: route fallback used"} {
		if !strings.Contains(line, expected) {
			t.Fatalf("expected %q in result status %q", expected, line)
		}
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
			summary: "cited 0/1 source",
			sources: 1,
		},
		{
			name:    "line-cited",
			result:  assistantSvc.Result{Message: "See README.md:12.", SourcePaths: []string{"README.md"}},
			quality: "line-cited",
			summary: "cited 1/1 source",
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

func TestAssistantEvidenceDiagnosticReportsPartialCitationCoverage(t *testing.T) {
	result := assistantSvc.Result{
		Message:     "Use README.md:12 for setup.",
		SourcePaths: []string{"README.md", "docs/guide.md", "notes/todo.md"},
	}
	diagnostic := assistantEvidenceDiagnosticForResult(result)
	if diagnostic.Quality != "line-cited" || diagnostic.CitedSourceCount != 1 || diagnostic.SourceCount != 3 {
		t.Fatalf("unexpected coverage diagnostic: %#v", diagnostic)
	}
	if strings.Join(diagnostic.CitedSourcePaths, "|") != "README.md" {
		t.Fatalf("unexpected cited sources: %#v", diagnostic.CitedSourcePaths)
	}
	if strings.Join(diagnostic.UncitedSourcePaths, "|") != "docs/guide.md|notes/todo.md" {
		t.Fatalf("unexpected uncited sources: %#v", diagnostic.UncitedSourcePaths)
	}
	for _, expected := range []string{"cited 1/3 source(s)", "uncited: docs/guide.md, notes/todo.md"} {
		if !strings.Contains(diagnostic.Summary, expected) {
			t.Fatalf("expected coverage summary to contain %q, got %q", expected, diagnostic.Summary)
		}
	}
}

func TestAssistantVisibleSourceDigestSummarizesLatestAnswer(t *testing.T) {
	empty := assistantVisibleSourceDigest(assistantSvc.Result{})
	if empty != "Source digest: no answer yet." {
		t.Fatalf("unexpected empty digest: %q", empty)
	}

	digest := assistantVisibleSourceDigest(assistantSvc.Result{
		Message:     "See README.md:12.",
		SourcePaths: []string{"README.md", "docs/guide.md"},
	})
	for _, expected := range []string{"Source digest:", "line-cited", "Sources: 2", "Verified refs: 1", "Unverified refs: 0"} {
		if !strings.Contains(digest, expected) {
			t.Fatalf("expected digest to contain %q, got %q", expected, digest)
		}
	}
}

func TestAssistantCitationSourceCoverageTreatsDirectorySourceAsCovered(t *testing.T) {
	cited, uncited := assistantCitationSourceCoverage([]string{"docs", "README.md"}, []string{"docs/guide.md:L4"})
	if strings.Join(cited, "|") != "docs" {
		t.Fatalf("expected docs directory to be covered, got cited=%#v uncited=%#v", cited, uncited)
	}
	if strings.Join(uncited, "|") != "README.md" {
		t.Fatalf("expected README to be uncited, got %#v", uncited)
	}
}

func TestAssistantSourcePathsFromContextKeepsLegacyRules(t *testing.T) {
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

func TestAssistantActionableSourcePathsClipsLargeSourceSets(t *testing.T) {
	paths := assistantActionableSourcePaths(assistantSvc.Result{
		SourcePaths: []string{"a.go", "b.go", "c.go", "d.go"},
	}, 2)
	if strings.Join(paths, "|") != "a.go|b.go" {
		t.Fatalf("unexpected clipped sources: %#v", paths)
	}
	all := assistantActionableSourcePaths(assistantSvc.Result{SourcePaths: []string{"a.go", "b.go"}}, 0)
	if strings.Join(all, "|") != "a.go|b.go" {
		t.Fatalf("expected non-positive limit to keep all sources, got %#v", all)
	}
}

func TestAssistantSourceActionSummaryReportsFailures(t *testing.T) {
	summary := assistantSourceActionSummary("Opened", 2, 4, 1)
	for _, expected := range []string{"Opened 2/4 assistant source(s).", "1 failed."} {
		if !strings.Contains(summary, expected) {
			t.Fatalf("expected %q in summary %q", expected, summary)
		}
	}
}

func TestAssistantSourceDigestMarkdownSummarizesEvidence(t *testing.T) {
	markdown := assistantSourceDigestMarkdown(assistantSvc.Result{
		Message:      "Use README.md:12 and missing.md:3.",
		Model:        "qwen",
		ModelRoute:   "Main coding model",
		SourcePaths:  []string{"README.md", "docs/guide.md"},
		RouteWarning: "fallback used",
	})
	for _, expected := range []string{
		"# Assistant Source Digest",
		"Evidence: line-cited",
		"Sources: 2. Verified refs: 1. Unverified refs: 1.",
		"Model: `qwen`",
		"Route: `Main coding model`",
		"Route warning: fallback used",
		"## Sources",
		"- `README.md`",
		"- `docs/guide.md`",
		"## Verified citations",
		"- `README.md:L12`",
		"## Unverified citations",
		"- `missing.md:L3`",
		"## Cited sources",
		"## Uncited sources",
	} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("expected digest to contain %q, got %q", expected, markdown)
		}
	}
}

func TestAssistantMarkdownListClipsLongLists(t *testing.T) {
	list := assistantMarkdownList("Sources", []string{"a.go", "b.go", "c.go"}, "empty", 2)
	if !strings.Contains(list, "- `a.go`") || !strings.Contains(list, "- `b.go`") || strings.Contains(list, "- `c.go`") || !strings.Contains(list, "1 more item(s) hidden") {
		t.Fatalf("unexpected clipped list: %q", list)
	}
	empty := assistantMarkdownList("Sources", nil, "No sources.", 2)
	if !strings.Contains(empty, "## Sources") || !strings.Contains(empty, "No sources.") {
		t.Fatalf("unexpected empty list: %q", empty)
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

func TestAssistantModelRouteOptionsIncludeGlobalFallbackAndRoutes(t *testing.T) {
	store := shellSettingsStore{settings: settingsSvc.Settings{
		ModelRoutes: []settingsSvc.ModelRoute{
			{ID: settingsSvc.RouteMainCoding, Label: "Main coding model", Model: "qwen3-coder:30b"},
		},
	}}
	options := assistantModelRouteOptions(store)
	if len(options) != 3 || options[0] != assistantAutoModelRouteLabel || options[1] != assistantGlobalModelRouteLabel || options[2] != "Main coding model" {
		t.Fatalf("unexpected route options: %#v", options)
	}
	if got := assistantModelRouteIDFromOption("Main coding model", store.settings.ModelRoutes); got != settingsSvc.RouteMainCoding {
		t.Fatalf("expected route id lookup, got %q", got)
	}
	if got := assistantModelRouteIDFromOption(assistantAutoModelRouteLabel, store.settings.ModelRoutes); got != "" {
		t.Fatalf("expected auto route option to defer inference, got %q", got)
	}
	if got := assistantModelRouteIDFromOption(assistantGlobalModelRouteLabel, store.settings.ModelRoutes); got != "" {
		t.Fatalf("expected global fallback to return empty route id, got %q", got)
	}
}

func TestInferAssistantModelRouteIDUsesPinnedContextFirst(t *testing.T) {
	got := inferAssistantModelRouteID("Summarize this screenshot", []string{"data/sales.csv"}, "mockup.png")
	if got != settingsSvc.RouteCSVExcelScripts {
		t.Fatalf("expected pinned data context to win, got %q", got)
	}
}

func TestInferAssistantModelRouteIDCoversDataDocumentVisionAndCode(t *testing.T) {
	cases := []struct {
		name     string
		prompt   string
		selected string
		want     string
	}{
		{name: "vision file", selected: "screens/home.png", want: settingsSvc.RouteVisionScreenshot},
		{name: "spreadsheet file", selected: "exports/sales.xlsx", want: settingsSvc.RouteCSVExcelScripts},
		{name: "sql file", selected: "queries/report.sql", want: settingsSvc.RouteSQL},
		{name: "document file", selected: "docs/report.pdf", want: settingsSvc.RouteResearchSummaries},
		{name: "go file", selected: "internal/app/main.go", want: settingsSvc.RouteGoBackend},
		{name: "prompt fallback", prompt: "Explain this PostgreSQL query", want: settingsSvc.RouteSQL},
		{name: "no signal", prompt: "Hello", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := inferAssistantModelRouteID(tc.prompt, nil, tc.selected); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestAssistantRouteBudgetLineShowsAutoRouteAndBudget(t *testing.T) {
	store := shellSettingsStore{settings: settingsSvc.Settings{
		ContextTokens:         1000,
		ResponseReserveTokens: 250,
		ModelRoutes: []settingsSvc.ModelRoute{{
			ID:                    settingsSvc.RouteCSVExcelScripts,
			Label:                 "CSV / Excel data scripts",
			Model:                 "qwen3-coder:30b",
			ContextTokens:         4000,
			ResponseReserveTokens: 1000,
		}},
	}}
	line := assistantRouteBudgetLine(store, assistantAutoModelRouteLabel, "", []string{"data/sales.csv"}, "")
	for _, expected := range []string{"Auto -> CSV / Excel data scripts", "Context budget", "11.7 KiB"} {
		if !strings.Contains(line, expected) {
			t.Fatalf("expected %q in budget line %q", expected, line)
		}
	}
}

func TestAssistantRouteBudgetLineFallsBackToSafeBudgetWhenReserveIsTooLarge(t *testing.T) {
	store := shellSettingsStore{settings: settingsSvc.Settings{
		ContextTokens:         1000,
		ResponseReserveTokens: 2000,
	}}
	line := assistantRouteBudgetLine(store, assistantGlobalModelRouteLabel, "", nil, "")
	if !strings.Contains(line, "Global fallback") || !strings.Contains(line, "96.0 KiB") {
		t.Fatalf("expected global safe fallback budget, got %q", line)
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
	if got := agentContextBudgetBytes(store, ""); got != 3000 {
		t.Fatalf("unexpected budget bytes: %d", got)
	}
}

func TestAgentContextBudgetBytesUsesSelectedModelRoute(t *testing.T) {
	store := shellSettingsStore{settings: settingsSvc.Settings{
		ContextTokens:         1000,
		ResponseReserveTokens: 250,
		ModelRoutes: []settingsSvc.ModelRoute{{
			ID:                    settingsSvc.RouteMainCoding,
			Label:                 "Main coding model",
			Model:                 "qwen3-coder:30b",
			ContextTokens:         4000,
			ResponseReserveTokens: 1000,
		}},
	}}
	if got := agentContextBudgetBytes(store, settingsSvc.RouteMainCoding); got != 12000 {
		t.Fatalf("unexpected routed budget bytes: %d", got)
	}
}

func TestAssistantAgentStatusLinesSummarizeJobSafetyAndResult(t *testing.T) {
	running := assistantAgentRunningStatusLine("job-1", agentSvc.Request{
		ModelRouteID:  settingsSvc.RouteMainCoding,
		SourcePaths:   []string{"README.md", "docs"},
		ApproveWrites: true,
		ApproveShell:  false,
	})
	for _, expected := range []string{"Running: Agent job job-1.", "Route: main-coding", "Sources: 2", "Writes: true", "Task tool: false", "Timeout: 45m"} {
		if !strings.Contains(running, expected) {
			t.Fatalf("expected %q in running status %q", expected, running)
		}
	}

	completed := assistantAgentResultStatusLine("job-1", agentSvc.Result{
		Model:      "qwen",
		ModelRoute: "Main coding model",
		Iterations: 3,
		ToolCalls:  []agentSvc.ToolResult{{Name: "read_context"}, {Name: "search_workspace"}},
		StopReason: "completed",
	})
	for _, expected := range []string{"Completed: Agent job job-1 with qwen via Main coding model", "3 iteration(s)", "2 tool call(s)", "Stop reason: completed."} {
		if !strings.Contains(completed, expected) {
			t.Fatalf("expected %q in completed status %q", expected, completed)
		}
	}
}

type shellSettingsStore struct {
	settings settingsSvc.Settings
}

func (s shellSettingsStore) Load() (settingsSvc.Settings, error) {
	return s.settings, nil
}

func (s shellSettingsStore) LoadForDisplay() (settingsSvc.Settings, error) {
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
