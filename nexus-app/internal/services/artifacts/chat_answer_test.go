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
		Prompt:           "Summarize README",
		Content:          "Use the setup guide.",
		Model:            "model-a",
		ContextRelPath:   "context: README.md",
		SourcePaths:      []string{"README.md"},
		CitationRefs:     []string{"README.md:L12"},
		CitationSnippets: []string{"README.md:L12 Third setup step."},
		EvidenceQuality:  "line-cited",
		EvidenceSummary:  "line-cited (1 source(s), 1 line ref(s)).",
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
	for _, expected := range []string{"# Assistant Answer - Summarize README", "Model:** model-a", "README.md", "## Citations", "README.md:L12", "## Citation Snippets", "Third setup step", "## Evidence", "Quality:** line-cited", "1 line ref", "## Prompt", "## Answer", "Use the setup guide."} {
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
	if len(metadata.CitationSnippets) != 1 || !strings.Contains(metadata.CitationSnippets[0], "Third setup step") {
		t.Fatalf("expected citation snippets in metadata, got %#v", metadata.CitationSnippets)
	}
	if metadata.EvidenceQuality != "line-cited" || !strings.Contains(metadata.EvidenceSummary, "1 line ref") {
		t.Fatalf("expected evidence metadata, got %#v", metadata)
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

func TestCleanChatAnswerSnippetsBoundsAndDedupes(t *testing.T) {
	snippets := []string{" same   snippet ", "same snippet"}
	for index := 0; index < chatAnswerCitationSnippetLimit+2; index++ {
		snippets = append(snippets, strings.Repeat("x", chatAnswerCitationSnippetMaxRunes+20))
	}
	cleaned := cleanChatAnswerSnippets(snippets)
	if len(cleaned) != 2 {
		t.Fatalf("expected deduped and bounded snippets, got %d: %#v", len(cleaned), cleaned)
	}
	if len([]rune(cleaned[1])) != chatAnswerCitationSnippetMaxRunes || !strings.HasSuffix(cleaned[1], "...") {
		t.Fatalf("expected long snippet to be truncated, got %q", cleaned[1])
	}
}
