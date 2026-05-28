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
	}, gitDiffModeUnified)

	for _, expected := range []string{
		"Staged diff / src/app.go",
		"Unstaged diff / src/app.go",
		"+staged",
		"+unstaged",
		"Diff output was windowed",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected formatted diff to contain %q, got:\n%s", expected, text)
		}
	}
}

func TestFormatGitDiffEmpty(t *testing.T) {
	text := formatGitDiff(gitSvc.FileDiff{Path: "README.md"}, gitDiffModeUnified)
	if text != "No staged or unstaged diff for README.md." {
		t.Fatalf("unexpected empty diff message: %q", text)
	}
}

func TestFormatGitDiffSplitMode(t *testing.T) {
	text := formatGitDiff(gitSvc.FileDiff{
		Path: "src/app.go",
		UnstagedDiff: strings.Join([]string{
			"diff --git a/src/app.go b/src/app.go",
			"@@ -1,3 +1,3 @@",
			" context",
			"-old",
			"+new",
		}, "\n"),
	}, gitDiffModeSplit)

	for _, expected := range []string{
		"Old\tNew",
		"context\tcontext",
		"old\t",
		"\tnew",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected split diff to contain %q, got:\n%s", expected, text)
		}
	}
}

func TestFormatGitDiffDiffOnlyModeSkipsContext(t *testing.T) {
	text := formatGitDiff(gitSvc.FileDiff{
		Path: "src/app.go",
		UnstagedDiff: strings.Join([]string{
			"diff --git a/src/app.go b/src/app.go",
			"@@ -1,3 +1,3 @@",
			" unchanged",
			"-old",
			"+new",
			" still unchanged",
		}, "\n"),
	}, gitDiffModeDiffOnly)

	if strings.Contains(text, "unchanged") {
		t.Fatalf("expected diff-only output to skip context, got:\n%s", text)
	}
	if !strings.Contains(text, "old\tnew") {
		t.Fatalf("expected changed lines side by side, got:\n%s", text)
	}
}

func TestGitHunkTargetsPreserveStagedThenUnstagedOrder(t *testing.T) {
	targets := gitHunkTargets(gitSvc.FileDiff{
		StagedHunks: []gitSvc.DiffHunk{
			{Kind: gitSvc.DiffKindStaged, Index: 0, Header: "@@ -1 +1 @@", AddedLines: 1, DeletedLines: 1},
		},
		UnstagedHunks: []gitSvc.DiffHunk{
			{Kind: gitSvc.DiffKindUnstaged, Index: 0, Header: "@@ -5 +5 @@", AddedLines: 2},
		},
	})

	if len(targets) != 2 {
		t.Fatalf("expected two hunk targets, got %#v", targets)
	}
	if targets[0].Kind != gitSvc.DiffKindStaged || targets[0].Label != "Staged hunk 1 (+1/-1)" {
		t.Fatalf("unexpected staged target: %#v", targets[0])
	}
	if targets[1].Kind != gitSvc.DiffKindUnstaged || targets[1].Label != "Unstaged hunk 1 (+2/-0)" {
		t.Fatalf("unexpected unstaged target: %#v", targets[1])
	}
}

func TestGitAIPromptsIncludeDiffAndPath(t *testing.T) {
	diff := gitSvc.FileDiff{
		Path:         "src/app.go",
		UnstagedDiff: "diff --git a/src/app.go b/src/app.go\n@@ -1 +1 @@\n-old\n+new\n",
	}
	summaryPrompt := gitSummaryPrompt(diff)
	commitPrompt := gitCommitDraftPrompt(diff)
	for _, prompt := range []string{summaryPrompt, commitPrompt} {
		if !strings.Contains(prompt, "Path: src/app.go") || !strings.Contains(prompt, "+new") {
			t.Fatalf("prompt did not include selected diff context:\n%s", prompt)
		}
	}
	if !strings.Contains(summaryPrompt, "risk") {
		t.Fatalf("summary prompt should ask for risk review: %q", summaryPrompt)
	}
	if !strings.Contains(commitPrompt, "commit message") {
		t.Fatalf("commit prompt should ask for a commit message: %q", commitPrompt)
	}
}

func TestGitAIDiffBlockTruncatesLargeDiff(t *testing.T) {
	diff := gitSvc.FileDiff{
		Path:         "large.txt",
		UnstagedDiff: strings.Repeat("+changed\n", maxGitAIDiffChars),
	}
	block := gitAIDiffBlock(diff)
	if len(block) > maxGitAIDiffChars+2000 {
		t.Fatalf("expected bounded AI diff block, got %d characters", len(block))
	}
	if !strings.Contains(block, "truncated") {
		t.Fatalf("expected truncation warning, got:\n%s", block)
	}
}
