package shell

import (
	"strings"
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

func TestFormatGitDiffIncludesSectionsAndTruncation(t *testing.T) {
	text := formatGitDiff(gitSvc.FileDiff{
		Path:                  "src/app.go",
		StagedDiff:            "diff --git a/src/app.go b/src/app.go\n+staged",
		UnstagedDiff:          "diff --git a/src/app.go b/src/app.go\n+unstaged",
		UnstagedDiffTruncated: true,
	})

	for _, expected := range []string{
		"Staged diff / src/app.go",
		"Unstaged diff / src/app.go",
		"+staged",
		"+unstaged",
		"Diff output was truncated.",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected formatted diff to contain %q, got:\n%s", expected, text)
		}
	}
}

func TestFormatGitDiffEmpty(t *testing.T) {
	text := formatGitDiff(gitSvc.FileDiff{Path: "README.md"})
	if text != "No staged or unstaged diff for README.md." {
		t.Fatalf("unexpected empty diff message: %q", text)
	}
}
