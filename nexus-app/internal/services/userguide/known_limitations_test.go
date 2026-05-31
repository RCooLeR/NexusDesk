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
		"Linux clean-machine trust validation",
		"SBOM",
		"Provider And Model Setup",
		"source warnings",
		"Agent And Tools",
		"Planned tools remain roadmap-only",
		"Data And Connectors",
		"read-only inspection",
		"Platform Coverage",
		"CI package smoke covers Windows, macOS, and Linux artifacts",
		"Secret Service",
	} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("expected %q in known limitations markdown:\n%s", expected, markdown)
		}
	}
}
