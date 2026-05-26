package storage

import (
	"path/filepath"
	"testing"
)

func TestChatHistoryStoreAppendsAndListsWorkspaceMessages(t *testing.T) {
	store := NewChatHistoryStore(filepath.Join(t.TempDir(), "chat.json"))
	workspace := filepath.Join(t.TempDir(), "workspace")

	messages, err := store.AppendPair(workspace, ChatMessage{Content: "Question"}, ChatMessage{Content: "Answer"})
	if err != nil {
		t.Fatalf("AppendPair failed: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Role != "user" || messages[1].Role != "assistant" {
		t.Fatalf("unexpected roles: %#v", messages)
	}
	if messages[0].CreatedAt == "" || messages[1].CreatedAt == "" {
		t.Fatal("expected timestamps")
	}
	if messages[0].CreatedAt == messages[1].CreatedAt {
		t.Fatalf("expected unique message timestamps, got %q", messages[0].CreatedAt)
	}

	read, err := store.List(workspace)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(read) != 2 {
		t.Fatalf("expected persisted messages, got %d", len(read))
	}
}

func TestChatHistoryStoreDisambiguatesProvidedPairTimestamps(t *testing.T) {
	store := NewChatHistoryStore(filepath.Join(t.TempDir(), "chat.json"))
	workspace := filepath.Join(t.TempDir(), "workspace")
	createdAt := "2026-05-25T12:00:00Z"

	messages, err := store.AppendPair(
		workspace,
		ChatMessage{Content: "Question", CreatedAt: createdAt},
		ChatMessage{Content: "Answer", CreatedAt: createdAt},
	)
	if err != nil {
		t.Fatalf("AppendPair failed: %v", err)
	}
	if messages[0].CreatedAt == messages[1].CreatedAt {
		t.Fatalf("expected unique timestamps, got %#v", messages)
	}
}

func TestChatHistoryStoreSeparatesWorkspaces(t *testing.T) {
	store := NewChatHistoryStore(filepath.Join(t.TempDir(), "chat.json"))
	first := filepath.Join(t.TempDir(), "first")
	second := filepath.Join(t.TempDir(), "second")

	if _, err := store.AppendPair(first, ChatMessage{Content: "First"}, ChatMessage{Content: "Answer"}); err != nil {
		t.Fatalf("AppendPair first failed: %v", err)
	}
	if _, err := store.AppendPair(second, ChatMessage{Content: "Second"}, ChatMessage{Content: "Answer"}); err != nil {
		t.Fatalf("AppendPair second failed: %v", err)
	}

	messages, err := store.List(first)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if messages[0].Content != "First" {
		t.Fatalf("expected first workspace history, got %#v", messages)
	}
}

func TestChatHistoryStoreLimitsMessages(t *testing.T) {
	store := NewChatHistoryStore(filepath.Join(t.TempDir(), "chat.json"))
	workspace := filepath.Join(t.TempDir(), "workspace")

	for index := 0; index < 60; index++ {
		if _, err := store.AppendPair(workspace, ChatMessage{Content: "Question"}, ChatMessage{Content: "Answer"}); err != nil {
			t.Fatalf("AppendPair failed: %v", err)
		}
	}

	messages, err := store.List(workspace)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(messages) != chatHistoryLimit {
		t.Fatalf("expected %d messages, got %d", chatHistoryLimit, len(messages))
	}
}

func TestChatHistoryStoreClearsWorkspace(t *testing.T) {
	store := NewChatHistoryStore(filepath.Join(t.TempDir(), "chat.json"))
	workspace := filepath.Join(t.TempDir(), "workspace")

	if _, err := store.AppendPair(workspace, ChatMessage{Content: "Question"}, ChatMessage{Content: "Answer"}); err != nil {
		t.Fatalf("AppendPair failed: %v", err)
	}

	messages, err := store.Clear(workspace)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected cleared messages, got %d", len(messages))
	}
}

func TestChatHistoryStoreSearchFindsMessages(t *testing.T) {
	store := NewChatHistoryStore(filepath.Join(t.TempDir(), "chat.json"))
	workspace := filepath.Join(t.TempDir(), "workspace")
	if _, err := store.AppendPair(workspace, ChatMessage{Content: "How is revenue?"}, ChatMessage{Content: "Revenue is up", SourcePaths: []string{"data/leads.csv"}}); err != nil {
		t.Fatalf("AppendPair returned error: %v", err)
	}

	results, err := store.Search(workspace, "revenue")
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 matching messages, got %d", len(results))
	}
}
