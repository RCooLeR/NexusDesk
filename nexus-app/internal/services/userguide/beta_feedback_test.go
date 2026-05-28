package userguide

import (
	"strings"
	"testing"
)

func TestBetaFeedbackGuideCoversPrivateBetaLoop(t *testing.T) {
	markdown := BetaFeedbackMarkdown()
	for _, expected := range []string{
		"Beta Feedback And Release Notes",
		"Before Reporting",
		"Redacted Issue Reports",
		"Release Notes",
		"Beta Feedback Loop",
		"app version",
		"commit",
		"build date",
		"Do not include API keys",
		"misleading assistant citations",
	} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("expected %q in beta feedback markdown:\n%s", expected, markdown)
		}
	}
}
