package artifacts

import (
	"os"
	"strings"
	"testing"
)

func TestWriteChatAnswerCreatesMarkdownAndMetadata(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	artifact, err := store.WriteChatAnswer(ChatAnswerReport{
		Prompt:         "Summarize README",
		Content:        "Use the setup guide.",
		Model:          "model-a",
		ContextRelPath: "context: README.md",
		SourcePaths:    []string{"README.md"},
		CitationRefs:   []string{"README.md:L12"},
	})
	if err != nil {
		t.Fatalf("WriteChatAnswer returned error: %v", err)
	}
	if artifact.Kind != "chat-answer" || !strings.Contains(artifact.RelPath, "/chat-answers/") {
		t.Fatalf("unexpected artifact: %#v", artifact)
	}
	data, err := os.ReadFile(artifact.AbsPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, expected := range []string{"# Assistant Answer - Summarize README", "Model:** model-a", "README.md", "## Citations", "README.md:L12", "## Prompt", "## Answer", "Use the setup guide."} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected artifact markdown to contain %q, got %q", expected, text)
		}
	}
	metadata, _ := store.readMetadata(artifact.RelPath)
	if metadata.Kind != "chat-answer" || metadata.Prompt != "Summarize README" || metadata.Model != "model-a" || metadata.ContextRelPath == "" {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
	if len(metadata.CitationRefs) != 1 || metadata.CitationRefs[0] != "README.md:L12" {
		t.Fatalf("expected citation refs in metadata, got %#v", metadata.CitationRefs)
	}
}

func TestWriteChatAnswerRequiresContent(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.WriteChatAnswer(ChatAnswerReport{Prompt: "Q"}); err == nil {
		t.Fatal("expected missing content error")
	}
}
