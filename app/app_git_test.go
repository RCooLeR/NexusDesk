package main

import "testing"

func TestParseGitStatus(t *testing.T) {
	changes, aheadBehind := parseGitStatus("## main...origin/main [ahead 1]\n M app.go\nA  new.go\nR  old.go -> next.go\n?? notes.md\n")
	if aheadBehind != "ahead 1" {
		t.Fatalf("unexpected ahead/behind: %q", aheadBehind)
	}
	if len(changes) != 4 {
		t.Fatalf("expected four changes, got %#v", changes)
	}
	if changes[0].Path != "app.go" || changes[0].Summary != "modified" {
		t.Fatalf("unexpected modified change: %#v", changes[0])
	}
	if changes[2].OldPath != "old.go" || changes[2].Path != "next.go" || changes[2].Summary != "renamed" {
		t.Fatalf("unexpected rename change: %#v", changes[2])
	}
	if changes[3].Summary != "untracked" {
		t.Fatalf("unexpected untracked change: %#v", changes[3])
	}
}

func TestSplitGitChanges(t *testing.T) {
	changes := []GitFileChange{
		{Path: "staged.go", Index: "M"},
		{Path: "unstaged.go", Worktree: "M"},
		{Path: "both.go", Index: "M", Worktree: "M"},
		{Path: "new.go", Index: "?", Worktree: "?"},
	}

	staged, unstaged := splitGitChanges(changes)
	if len(staged) != 2 {
		t.Fatalf("expected two staged changes, got %#v", staged)
	}
	if len(unstaged) != 3 {
		t.Fatalf("expected three unstaged changes, got %#v", unstaged)
	}
}
