package shell

import (
	"strings"
	"testing"
	"time"

	agentSvc "nexusdesk/internal/services/agent"
	llmSvc "nexusdesk/internal/services/llm"
	metadataSvc "nexusdesk/internal/services/metadata"
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
