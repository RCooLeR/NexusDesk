package userguide

import (
	"strings"
	"testing"
)

func TestKnownLimitationsGuideCoversBetaBoundaries(t *testing.T) {
	markdown := KnownLimitationsMarkdown()
	for _, expected := range []string{
		"Known Limitations",
		"Packaging And Trust",
		"Windows signing",
		"SBOM",
		"Provider And Model Setup",
		"source warnings",
		"Agent And Tools",
		"Planned tools remain roadmap-only",
		"Data And Connectors",
		"read-only inspection",
		"Platform Coverage",
		"Secret Service",
	} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("expected %q in known limitations markdown:\n%s", expected, markdown)
		}
	}
}
