package userguide

import (
	"strings"
	"testing"
)

func TestUpdateCheckMarkdownExplainsManualPolicy(t *testing.T) {
	markdown := UpdateCheckMarkdown("1.2.3", "abcdef", "2026-05-30T12:00:00Z")
	for _, expected := range []string{
		"Check For Updates",
		"Version: 1.2.3",
		"Commit: abcdef",
		"Build date: 2026-05-30T12:00:00Z",
		"does not download update artifacts automatically",
		"does not install updates automatically",
		"docs/releases/beta-release-notes.md",
	} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("expected update guidance to contain %q:\n%s", expected, markdown)
		}
	}
}
