package shell

import (
	"testing"

	gitSvc "nexusdesk/internal/services/git"
)

func TestGroupGitChangesSortsByDirectory(t *testing.T) {
	groups := groupGitChanges([]gitSvc.FileChange{
		{Path: "src/z.go", Summary: "modified"},
		{Path: "README.md", Summary: "modified"},
		{Path: "src/a.go", Summary: "added"},
		{Path: "docs/guide.md", Summary: "deleted"},
	})

	if len(groups) != 3 {
		t.Fatalf("expected three groups, got %#v", groups)
	}
	if groups[0].Directory != "Workspace root" || groups[0].Changes[0].Path != "README.md" {
		t.Fatalf("expected root group first, got %#v", groups)
	}
	if groups[1].Directory != "docs" {
		t.Fatalf("expected docs group second, got %#v", groups)
	}
	if groups[2].Directory != "src" || groups[2].Changes[0].Path != "src/a.go" || groups[2].Changes[1].Path != "src/z.go" {
		t.Fatalf("expected sorted src changes, got %#v", groups[2])
	}
}
