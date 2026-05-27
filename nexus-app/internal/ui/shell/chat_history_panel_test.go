package shell

import (
	"strings"
	"testing"
	"time"

	metadataSvc "nexusdesk/internal/services/metadata"
)

func TestChatHistoryFormatting(t *testing.T) {
	record := metadataSvc.ChatMessageRecord{
		Role:        "assistant",
		Content:     "Answer body",
		Model:       "qwen",
		SourcePaths: []string{"README.md", "tracker.md"},
		CreatedAt:   time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC),
	}
	if got := chatHistoryStatusText("native", 3); got != `Chat history: 3 result(s) for "native".` {
		t.Fatalf("unexpected status: %q", got)
	}
	if got := chatHistoryRowTitle(record); got != "Assistant" {
		t.Fatalf("unexpected title: %q", got)
	}
	detail := formatChatHistoryRecord(record)
	for _, want := range []string{"Assistant", "qwen", "Sources: README.md, tracker.md", "Answer body"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("expected detail to contain %q:\n%s", want, detail)
		}
	}
}

func TestCompactChatHistoryContent(t *testing.T) {
	got := compactChatHistoryContent("one\n\n two   three four", 13)
	if got != "one two th..." {
		t.Fatalf("unexpected compact content: %q", got)
	}
}
