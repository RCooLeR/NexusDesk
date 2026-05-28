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
		Prompt:                 "Summarize README",
		Content:                "Use the setup guide.",
		Model:                  "model-a",
		ModelRouteID:           "main-coding",
		ModelRoute:             "Main coding model",
		ContextRelPath:         "context: README.md",
		SourcePaths:            []string{"README.md", "docs/guide.md"},
		CitationRefs:           []string{"README.md:L12"},
		UnverifiedCitationRefs: []string{"other.md:L3"},
		CitationSnippets:       []string{"README.md:L12 Third setup step."},
		CitedSourcePaths:       []string{"README.md"},
		UncitedSourcePaths:     []string{"docs/guide.md"},
		EvidenceQuality:        "line-cited",
		EvidenceSummary:        "line-cited (2 source(s), 1 line ref(s), cited 1/2 source(s); uncited: docs/guide.md; 1 citation outside selected sources).",
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
	for _, expected := range []string{"# Assistant Answer - Summarize README", "Model:** model-a", "Model route:** Main coding model", "README.md", "## Citations", "README.md:L12", "## Unverified Citations", "other.md:L3", "## Citation Snippets", "Third setup step", "## Evidence", "Quality:** line-cited", "outside selected sources", "## Source Coverage", "Cited sources:** 1", "Uncited sources:** 1", "## Prompt", "## Answer", "Use the setup guide."} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected artifact markdown to contain %q, got %q", expected, text)
		}
	}
	metadata, _ := store.readMetadata(artifact.RelPath)
	if metadata.Kind != "chat-answer" || metadata.Prompt != "Summarize README" || metadata.Model != "model-a" || metadata.ContextRelPath == "" {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
	if metadata.ModelRouteID != "main-coding" || metadata.ModelRoute != "Main coding model" {
		t.Fatalf("expected model route metadata, got %#v", metadata)
	}
	if len(metadata.CitationRefs) != 1 || metadata.CitationRefs[0] != "README.md:L12" {
		t.Fatalf("expected citation refs in metadata, got %#v", metadata.CitationRefs)
	}
	if len(metadata.UnverifiedCitationRefs) != 1 || metadata.UnverifiedCitationRefs[0] != "other.md:L3" {
		t.Fatalf("expected unverified citation refs in metadata, got %#v", metadata.UnverifiedCitationRefs)
	}
	if len(metadata.CitationSnippets) != 1 || !strings.Contains(metadata.CitationSnippets[0], "Third setup step") {
		t.Fatalf("expected citation snippets in metadata, got %#v", metadata.CitationSnippets)
	}
	if len(metadata.CitedSourcePaths) != 1 || metadata.CitedSourcePaths[0] != "README.md" {
		t.Fatalf("expected cited source coverage in metadata, got %#v", metadata.CitedSourcePaths)
	}
	if len(metadata.UncitedSourcePaths) != 1 || metadata.UncitedSourcePaths[0] != "docs/guide.md" {
		t.Fatalf("expected uncited source coverage in metadata, got %#v", metadata.UncitedSourcePaths)
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

func TestExtractChatAnswerContentUsesAnswerSection(t *testing.T) {
	content := ExtractChatAnswerContent("# Assistant Answer\n\n## Prompt\n\nQ\n\n## Answer\n\nA\n\n## Sources\n\n- README.md\n")
	if content != "A" {
		t.Fatalf("unexpected extracted answer content: %q", content)
	}
}

func TestExtractChatAnswerContentFallsBackToTrimmedMarkdown(t *testing.T) {
	content := ExtractChatAnswerContent("\n# Legacy Answer\n\nUse the README.\n")
	if content != "# Legacy Answer\n\nUse the README." {
		t.Fatalf("unexpected legacy content: %q", content)
	}
}
