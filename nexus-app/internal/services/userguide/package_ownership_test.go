package userguide

import (
	"strings"
	"testing"
)

func TestPackageOwnershipGuideCoversMajorInternalAreas(t *testing.T) {
	markdown := PackageOwnershipMarkdown()
	for _, expected := range []string{
		"Internal Package Ownership",
		"`internal/app`",
		"`internal/domain`",
		"`internal/services`",
		"`internal/ui`",
		"`services/workspace`",
		"`services/metadata`",
		"`services/jobs`",
		"`services/artifacts`",
		"`services/llm`",
		"`internal/ui/shell`",
		"`internal/architecture`",
		"`internal/release`",
		"Fyne",
		"Wails/webview",
		"folder open cheap",
		"approvals, audit, redaction",
	} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("expected %q in package ownership markdown:\n%s", expected, markdown)
		}
	}
}

func TestPackageOwnershipGuideKeepsLayerRulesFirst(t *testing.T) {
	guide := PackageOwnershipGuide()
	if guide.Title == "" || len(guide.Sections) < 6 {
		t.Fatalf("expected complete package ownership guide, got %#v", guide)
	}
	if guide.Sections[0].Title != "Layer Rules" {
		t.Fatalf("expected layer rules first, got %q", guide.Sections[0].Title)
	}
}
