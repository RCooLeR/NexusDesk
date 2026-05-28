package userguide

import (
	"strings"
	"testing"
)

func TestSafeAgentGuideCoversReleaseSafetyTopics(t *testing.T) {
	guide := SafeAgentGuide()
	if guide.Title == "" || len(guide.Sections) < 6 {
		t.Fatalf("expected complete safe-agent guide, got %#v", guide)
	}
	markdown := SafeAgentMarkdown()
	for _, expected := range []string{
		"Approvals And Risky Actions",
		"Rollbacks And Recovery",
		"Local Data And Secrets",
		"Connectors And Databases",
		"Slow Work And Jobs",
		"redact",
		"protected OS storage",
	} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("expected %q in safe-agent markdown:\n%s", expected, markdown)
		}
	}
}

func TestFormatMarkdownSkipsEmptySections(t *testing.T) {
	markdown := FormatMarkdown(Guide{
		Title:   "Guide",
		Summary: "Summary",
		Sections: []Section{
			{Title: "Keep", Body: []string{"Use approvals.", ""}},
			{Title: "   ", Body: []string{"drop"}},
		},
	})
	if strings.Contains(markdown, "drop") {
		t.Fatalf("expected empty-title section to be skipped:\n%s", markdown)
	}
	if !strings.Contains(markdown, "- Use approvals.") {
		t.Fatalf("expected non-empty paragraph to render:\n%s", markdown)
	}
	if !strings.HasSuffix(markdown, "\n") {
		t.Fatalf("expected markdown to end with newline")
	}
}
