package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	metadataSvc "nexusdesk/internal/services/metadata"
)

func TestChatHistoryFormatting(t *testing.T) {
	record := metadataSvc.ChatMessageRecord{
		Role:           "assistant",
		Content:        "Answer body",
		Model:          "qwen",
		ContextRelPath: "context: README.md",
		SourcePaths:    []string{"README.md", "tracker.md"},
		CreatedAt:      time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC),
	}
	if got := chatHistoryStatusText("native", 3); got != `Chat history: 3 result(s) for "native".` {
		t.Fatalf("unexpected status: %q", got)
	}
	if got := chatHistoryRowTitle(record); got != "Assistant" {
		t.Fatalf("unexpected title: %q", got)
	}
	detail := formatChatHistoryRecord(record, []string{"README.md"})
	for _, want := range []string{"Assistant", "qwen", "Context: context: README.md", "Sources: README.md, tracker.md", "context changed since this answer was created", "Answer body"} {
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

func TestChatHistorySeedPromptIncludesRoleAndSourceHint(t *testing.T) {
	prompt := chatHistorySeedPrompt(metadataSvc.ChatMessageRecord{
		Role:        "assistant",
		Content:     "Prior answer",
		SourcePaths: []string{"README.md"},
	})
	for _, want := range []string{"prior assistant message", "Original source paths are pinned", "Prior answer", "Next task:"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected prompt to contain %q:\n%s", want, prompt)
		}
	}
}

func TestChatHistoryFreshnessFlagsChangedAndMissingSources(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "README.md")
	if err := os.WriteFile(sourcePath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	created := time.Now().UTC().Add(-2 * time.Hour)
	newer := time.Now().UTC()
	if err := os.Chtimes(sourcePath, newer, newer); err != nil {
		t.Fatal(err)
	}
	record := metadataSvc.ChatMessageRecord{
		ID:          "chat-1",
		Role:        "assistant",
		Content:     "Answer",
		SourcePaths: []string{"README.md", "missing.md", "../ignored.txt"},
		CreatedAt:   created,
	}
	freshness := chatHistoryFreshness(root, []metadataSvc.ChatMessageRecord{record})
	stale := strings.Join(freshness[record.ID], ",")
	if !strings.Contains(stale, "README.md") || !strings.Contains(stale, "missing.md") || strings.Contains(stale, "ignored") {
		t.Fatalf("unexpected stale sources: %#v", freshness[record.ID])
	}
}
