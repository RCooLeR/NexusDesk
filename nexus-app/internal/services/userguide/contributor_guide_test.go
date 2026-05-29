package userguide

import (
	"strings"
	"testing"
)

func TestContributorGuideCoversSetupStandardsAndADR(t *testing.T) {
	markdown := ContributorMarkdown()
	for _, expected := range []string{
		"Contributor Setup And Standards",
		"`nexus-app/`",
		"`nexus-app/`",
		"MSYS2 UCRT64 GCC",
		"`GOFLAGS=-mod=readonly`",
		"Fyne imports",
		"folder open cheap",
		"`go test ./...`",
		"`git diff --check`",
		"`tracker.md`",
		"`docs/adr/NNNN-short-title.md`",
		"Context, Decision, Consequences, Status, and Date",
		"Do not revert unrelated user changes",
	} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("expected %q in contributor markdown:\n%s", expected, markdown)
		}
	}
}

func TestContributorGuideKeepsADRProcessVisible(t *testing.T) {
	guide := ContributorGuide()
	if guide.Title == "" || len(guide.Sections) < 7 {
		t.Fatalf("expected complete contributor guide, got %#v", guide)
	}
	foundADR := false
	for _, section := range guide.Sections {
		if section.Title == "ADR Process" {
			foundADR = true
			break
		}
	}
	if !foundADR {
		t.Fatalf("expected ADR Process section in %#v", guide.Sections)
	}
}
