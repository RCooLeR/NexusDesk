package shell

import (
	"testing"

	gitSvc "nexusdesk/internal/services/git"
)

func TestGitWorkspaceBadgesMarksChangedFilesAndParents(t *testing.T) {
	badges := gitWorkspaceBadges(gitSvc.Status{
		Available: true,
		ChangedFiles: []gitSvc.FileChange{
			{Path: "app/main.go", Worktree: "M"},
			{Path: "docs/new.md", Index: "?", Worktree: "?"},
			{Path: "old.txt", OldPath: "legacy.txt", Index: "R"},
		},
	})

	expected := map[string]string{
		"app":         gitChangedDirectoryBadge,
		"app/main.go": "M",
		"docs":        gitChangedDirectoryBadge,
		"docs/new.md": "?",
		"old.txt":     "R",
		"legacy.txt":  "R",
	}
	for path, badge := range expected {
		if badges[path] != badge {
			t.Fatalf("badge for %s = %q, want %q in %#v", path, badges[path], badge, badges)
		}
	}
}

func TestGitWorkspaceBadgesSkipUnavailableStatus(t *testing.T) {
	if badges := gitWorkspaceBadges(gitSvc.Status{}); len(badges) != 0 {
		t.Fatalf("expected no badges for unavailable status, got %#v", badges)
	}
}

func TestGitChangeBadgeFallbacks(t *testing.T) {
	cases := []struct {
		change gitSvc.FileChange
		want   string
	}{
		{change: gitSvc.FileChange{Index: "A"}, want: "A"},
		{change: gitSvc.FileChange{Worktree: "D"}, want: "D"},
		{change: gitSvc.FileChange{Index: "?", Worktree: "?"}, want: "?"},
		{change: gitSvc.FileChange{}, want: "!"},
	}
	for _, tt := range cases {
		if got := gitChangeBadge(tt.change); got != tt.want {
			t.Fatalf("gitChangeBadge(%#v) = %q, want %q", tt.change, got, tt.want)
		}
	}
}
