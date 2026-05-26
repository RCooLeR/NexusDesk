package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"NexusAugenticStudio/internal/storage"
	"NexusAugenticStudio/internal/workspace"
)

func TestBuildChatContextContentUsesCSVSummary(t *testing.T) {
	content := buildChatContextContent(workspace.FilePreview{
		Name:    "report.csv",
		Kind:    "file",
		Content: "raw,csv\nalpha,10\n",
		Table: &workspace.TablePreview{
			Columns: []string{"name", "value"},
			Rows: [][]string{
				{"alpha", "10"},
				{"beta, quoted", "20"},
			},
			Profiles: []workspace.ColumnProfile{
				{Name: "name", Type: "text", Missing: 0, Distinct: 2},
				{Name: "value", Type: "integer", Missing: 0, Distinct: 2, Min: "10", Max: "20"},
			},
		},
	})

	if !strings.Contains(content, "CSV context summary") {
		t.Fatalf("expected CSV context summary, got %q", content)
	}
	if !strings.Contains(content, "value: integer, distinct=2, missing=0, range=10..20") {
		t.Fatalf("expected numeric profile, got %q", content)
	}
	if !strings.Contains(content, "\"beta, quoted\",20") {
		t.Fatalf("expected CSV sample rows to stay escaped, got %q", content)
	}
}

func TestBuildChatContextContentKeepsTextContent(t *testing.T) {
	content := buildChatContextContent(workspace.FilePreview{
		Name:    "notes.md",
		Kind:    "file",
		Content: "plain text",
	})

	if content != "plain text" {
		t.Fatalf("expected plain text context, got %q", content)
	}
}

func TestCleanContextPathsDeduplicatesAndDropsEmptyValues(t *testing.T) {
	paths := cleanContextPaths([]string{"a.md", "", "a.md", " b.md "})

	if len(paths) != 2 || paths[0] != "a.md" || paths[1] != "b.md" {
		t.Fatalf("unexpected context paths: %#v", paths)
	}
}

func TestPrepareChatBuildsDirectoryContextPack(t *testing.T) {
	root := t.TempDir()
	writeAppTestFile(t, root, "src/main.go", "package main\n")
	writeAppTestFile(t, root, "src/readme.md", "# Source\n")

	app := NewApp()
	app.llmStore = storage.NewLLMSettingsStore(filepath.Join(t.TempDir(), "llm-settings.json"))
	app.setWorkspaceRoot(root)

	request, _, err := app.prepareChat("explain this folder", []string{"src"})
	if err != nil {
		t.Fatalf("prepareChat returned error: %v", err)
	}

	if !strings.HasPrefix(request.ContextRelPath, "dir: src") {
		t.Fatalf("expected directory context label, got %q", request.ContextRelPath)
	}
	if !strings.Contains(request.ContextContent, "Workspace context: src/main.go") {
		t.Fatalf("expected main.go in context pack, got %q", request.ContextContent)
	}
	if !strings.Contains(request.ContextContent, "Workspace context: src/readme.md") {
		t.Fatalf("expected readme.md in context pack, got %q", request.ContextContent)
	}
}

func TestPrepareChatBuildsProjectContextPack(t *testing.T) {
	root := t.TempDir()
	writeAppTestFile(t, root, "README.md", "# Project\n")
	writeAppTestFile(t, root, "app/main.go", "package main\n")

	app := NewApp()
	app.llmStore = storage.NewLLMSettingsStore(filepath.Join(t.TempDir(), "llm-settings.json"))
	app.setWorkspaceRoot(root)

	request, _, err := app.prepareChat("summarize project", []string{"."})
	if err != nil {
		t.Fatalf("prepareChat returned error: %v", err)
	}

	if !strings.HasPrefix(request.ContextRelPath, "project:") {
		t.Fatalf("expected project context label, got %q", request.ContextRelPath)
	}
	if !strings.Contains(request.ContextContent, "Requested roots: .") {
		t.Fatalf("expected project root manifest, got %q", request.ContextContent)
	}
}

func TestPrepareChatUsesConfiguredContextWindowBudget(t *testing.T) {
	root := t.TempDir()
	largeContent := strings.Repeat("a", 110*1024) + "\nTAIL-MARKER\n"
	writeAppTestFile(t, root, "large.md", largeContent)

	settingsPath := filepath.Join(t.TempDir(), "llm-settings.json")
	store := storage.NewLLMSettingsStore(settingsPath)
	if _, err := store.Save(storage.LLMSettings{
		ProviderName:          "Local OpenAI-compatible",
		BaseURL:               "http://localhost:11434/v1",
		Model:                 "qwen3:8b",
		MaxContextTokens:      65536,
		ResponseReserveTokens: 4096,
	}); err != nil {
		t.Fatalf("Save settings failed: %v", err)
	}

	app := NewApp()
	app.llmStore = store
	app.setWorkspaceRoot(root)

	request, _, err := app.prepareChat("use the full context", []string{"large.md"})
	if err != nil {
		t.Fatalf("prepareChat returned error: %v", err)
	}
	if !strings.Contains(request.ContextContent, "TAIL-MARKER") {
		t.Fatalf("expected configured context budget to include tail marker, got %d bytes", len(request.ContextContent))
	}
}

func TestPrepareChatIncludesRecentConversationHistory(t *testing.T) {
	root := t.TempDir()
	app := NewApp()
	app.llmStore = storage.NewLLMSettingsStore(filepath.Join(t.TempDir(), "llm-settings.json"))
	app.chatStore = storage.NewChatHistoryStore(filepath.Join(t.TempDir(), "chat.json"))
	app.setWorkspaceRoot(root)

	if _, err := app.chatStore.AppendPair(root, storage.ChatMessage{
		Role:    "user",
		Content: "What is this project?",
	}, storage.ChatMessage{
		Role:    "assistant",
		Content: "It is Nexus.",
	}); err != nil {
		t.Fatalf("AppendPair failed: %v", err)
	}

	request, _, err := app.prepareChat("continue", nil)
	if err != nil {
		t.Fatalf("prepareChat returned error: %v", err)
	}
	if len(request.Conversation) != 2 {
		t.Fatalf("expected two history turns, got %#v", request.Conversation)
	}
	if request.Conversation[0].Role != "user" || request.Conversation[0].Content != "What is this project?" {
		t.Fatalf("unexpected first history turn: %#v", request.Conversation[0])
	}
	if request.Conversation[1].Role != "assistant" || request.Conversation[1].Content != "It is Nexus." {
		t.Fatalf("unexpected second history turn: %#v", request.Conversation[1])
	}
}

func TestTruncateContextStringKeepsUTF8Valid(t *testing.T) {
	content := "prefix " + string('\u03C0')
	truncated := truncateContextString(content, len("prefix \u03C0")+1)

	if !utf8.ValidString(truncated) {
		t.Fatalf("expected valid UTF-8, got %q", truncated)
	}
	if len(truncated) > len("prefix \u03C0")+1 {
		t.Fatalf("expected byte cap to be respected, got %d", len(truncated))
	}
}

func writeAppTestFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}
