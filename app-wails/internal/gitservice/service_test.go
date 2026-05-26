package gitservice

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestParseGitStatus(t *testing.T) {
	changes, aheadBehind := parseStatus("## main...origin/main [ahead 1]\n M app.go\nA  new.go\nR  old.go -> next.go\n?? notes.md\n")
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
	changes := []FileChange{
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

func TestCleanGitRelPath(t *testing.T) {
	path, err := cleanGitRelPath(`"app/frontend/src/App.tsx"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "app/frontend/src/App.tsx" {
		t.Fatalf("unexpected path: %q", path)
	}

	for _, value := range []string{"", "..", "../outside.go", "app/../outside.go", "app/..", "-bad"} {
		if _, err := cleanGitRelPath(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestGitFileActionCommand(t *testing.T) {
	stage, err := gitFileActionCommand(gitFileActionStage, "app/main.go")
	if err != nil {
		t.Fatalf("unexpected stage error: %v", err)
	}
	if got := strings.Join(stage, " "); got != "git add -- app/main.go" {
		t.Fatalf("unexpected stage command: %q", got)
	}

	unstage, err := gitFileActionCommand(gitFileActionUnstage, "app/main.go")
	if err != nil {
		t.Fatalf("unexpected unstage error: %v", err)
	}
	if got := strings.Join(unstage, " "); got != "git restore --staged -- app/main.go" {
		t.Fatalf("unexpected unstage command: %q", got)
	}

	if _, err := gitFileActionCommand("discard", "app/main.go"); err == nil {
		t.Fatal("expected unsupported action to fail")
	}
}

func TestGitHunkActionCommand(t *testing.T) {
	stage, err := gitHunkActionCommand(gitHunkActionStage, gitDiffKindUnstaged)
	if err != nil {
		t.Fatalf("unexpected stage error: %v", err)
	}
	if got := strings.Join(stage, " "); got != "git apply --cached --whitespace=nowarn" {
		t.Fatalf("unexpected stage command: %q", got)
	}

	unstage, err := gitHunkActionCommand(gitHunkActionUnstage, gitDiffKindStaged)
	if err != nil {
		t.Fatalf("unexpected unstage error: %v", err)
	}
	if got := strings.Join(unstage, " "); got != "git apply --cached --reverse --whitespace=nowarn" {
		t.Fatalf("unexpected unstage command: %q", got)
	}

	discard, err := gitHunkActionCommand(gitHunkActionDiscard, gitDiffKindUnstaged)
	if err != nil {
		t.Fatalf("unexpected discard error: %v", err)
	}
	if got := strings.Join(discard, " "); got != "git apply --reverse --whitespace=nowarn" {
		t.Fatalf("unexpected discard command: %q", got)
	}

	revert, err := gitHunkActionCommand(gitHunkActionRevert, gitDiffKindStaged)
	if err != nil {
		t.Fatalf("unexpected revert error: %v", err)
	}
	if got := strings.Join(revert, " "); got != "git apply --cached --reverse --whitespace=nowarn" {
		t.Fatalf("unexpected revert command: %q", got)
	}

	if _, err := gitHunkActionCommand(gitHunkActionDiscard, gitDiffKindStaged); err == nil {
		t.Fatal("expected invalid hunk action/kind pair to fail")
	}
}

func TestParseGitHistory(t *testing.T) {
	output := strings.Join([]string{
		"1111111111111111111111111111111111111111\x1f1111111\x1fAda\x1fada@example.com\x1f2026-01-01T00:00:00Z\x1fAdd core",
		"2222222222222222222222222222222222222222\x1f2222222\x1fGrace\x1fgrace@example.com\x1f2026-01-02T00:00:00Z\x1fTune UI",
	}, "\n")
	entries, truncated := parseGitHistory(output, 1)
	if !truncated {
		t.Fatal("expected history to report truncation")
	}
	if len(entries) != 1 || entries[0].ShortHash != "1111111" || entries[0].Subject != "Add core" {
		t.Fatalf("unexpected history entries: %#v", entries)
	}
}

func TestParseGitBlame(t *testing.T) {
	output := strings.Join([]string{
		"1111111111111111111111111111111111111111 1 7 1",
		"author Ada",
		"author-time 1767225600",
		"summary Add core",
		"\tline seven",
	}, "\n")
	lines, truncated := parseGitBlame(output, 10)
	if truncated {
		t.Fatal("did not expect blame truncation")
	}
	if len(lines) != 1 {
		t.Fatalf("expected one blame line, got %#v", lines)
	}
	if lines[0].Line != 7 || lines[0].Author != "Ada" || lines[0].ShortHash != "111111111111" || lines[0].Content != "line seven" {
		t.Fatalf("unexpected blame line: %#v", lines[0])
	}
}

func TestApplyFileActionStagesFile(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := t.TempDir()
	runTestGit(t, root, "init")
	runTestGit(t, root, "config", "user.email", "test@example.com")
	runTestGit(t, root, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("initial\n"), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}
	runTestGit(t, root, "add", "notes.txt")
	runTestGit(t, root, "commit", "-m", "initial")
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("write changed file: %v", err)
	}

	service := Service{workspaceRoot: func() string { return root }}
	preview, err := service.ApplyFileAction(FileActionRequest{
		Path:   "notes.txt",
		Action: gitFileActionStage,
	})
	if err != nil {
		t.Fatalf("ApplyFileAction returned error: %v", err)
	}
	if preview.Status.Available != true {
		t.Fatalf("expected status in file action result: %#v", preview)
	}
	if preview.Message != "Applied stage for notes.txt." {
		t.Fatalf("unexpected message: %q", preview.Message)
	}
	status := runTestGitOutput(t, root, "status", "--porcelain=v1", "--", "notes.txt")
	if !strings.HasPrefix(status, "M ") {
		t.Fatalf("expected file to be staged, got %q", status)
	}
}

func TestServiceHistoryAndBlame(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := t.TempDir()
	runTestGit(t, root, "init")
	runTestGit(t, root, "config", "user.email", "test@example.com")
	runTestGit(t, root, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("line one\nline two\n"), 0o644); err != nil {
		t.Fatalf("write notes file: %v", err)
	}
	runTestGit(t, root, "add", "notes.txt")
	runTestGit(t, root, "commit", "-m", "initial notes")

	service := Service{workspaceRoot: func() string { return root }}
	history, err := service.History(HistoryRequest{Path: "notes.txt", Limit: 5})
	if err != nil {
		t.Fatalf("History returned error: %v", err)
	}
	if !history.Available || len(history.Entries) != 1 || history.Entries[0].Subject != "initial notes" {
		t.Fatalf("unexpected history result: %#v", history)
	}

	blame, err := service.Blame(BlameRequest{Path: "notes.txt", StartLine: 2, EndLine: 2})
	if err != nil {
		t.Fatalf("Blame returned error: %v", err)
	}
	if !blame.Available || len(blame.Lines) != 1 || blame.Lines[0].Line != 2 || blame.Lines[0].Content != "line two" {
		t.Fatalf("unexpected blame result: %#v", blame)
	}
}

func TestExtractGitHunkPatch(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/app.go b/app.go",
		"index 1111111..2222222 100644",
		"--- a/app.go",
		"+++ b/app.go",
		"@@ -1,3 +1,3 @@",
		" line 1",
		"-old 2",
		"+new 2",
		" line 3",
		"@@ -9,3 +9,3 @@",
		" line 9",
		"-old 10",
		"+new 10",
		" line 11",
	}, "\n")
	patch, err := extractGitHunkPatch(diff, 2)
	if err != nil {
		t.Fatalf("extractGitHunkPatch returned error: %v", err)
	}
	if !strings.Contains(patch, "--- a/app.go\n+++ b/app.go\n@@ -9,3 +9,3 @@") {
		t.Fatalf("patch did not include header and target hunk: %q", patch)
	}
	if strings.Contains(patch, "@@ -1,3 +1,3 @@") {
		t.Fatalf("patch included the wrong hunk: %q", patch)
	}
}

func TestApplyHunkActionDiscardsUnstagedHunk(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := t.TempDir()
	runTestGit(t, root, "init")
	runTestGit(t, root, "config", "user.email", "test@example.com")
	runTestGit(t, root, "config", "user.name", "Test User")
	initial := []string{}
	for index := 1; index <= 30; index++ {
		initial = append(initial, "line "+strconv.Itoa(index))
	}
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte(strings.Join(initial, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}
	runTestGit(t, root, "add", "notes.txt")
	runTestGit(t, root, "commit", "-m", "initial")

	modified := append([]string{}, initial...)
	modified[1] = "changed 2"
	modified[24] = "changed 25"
	if err := os.WriteFile(path, []byte(strings.Join(modified, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write modified file: %v", err)
	}

	service := Service{workspaceRoot: func() string { return root }}
	preview, err := service.ApplyHunkAction(HunkActionRequest{
		Path:      "notes.txt",
		Action:    gitHunkActionDiscard,
		DiffKind:  gitDiffKindUnstaged,
		HunkIndex: 1,
	})
	if err != nil {
		t.Fatalf("ApplyHunkAction returned error: %v", err)
	}
	if preview.Status.Available != true {
		t.Fatalf("expected status in hunk action result: %#v", preview)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read modified file: %v", err)
	}
	text := string(content)
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if lines[1] != "line 2" {
		t.Fatalf("expected first hunk to be discarded: %q", text)
	}
	if lines[24] != "changed 25" {
		t.Fatalf("expected second hunk to remain: %q", text)
	}
}

func TestApplyHunkActionStagesUnstagedHunk(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := t.TempDir()
	runTestGit(t, root, "init")
	runTestGit(t, root, "config", "user.email", "test@example.com")
	runTestGit(t, root, "config", "user.name", "Test User")
	initial := []string{}
	for index := 1; index <= 30; index++ {
		initial = append(initial, "line "+strconv.Itoa(index))
	}
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte(strings.Join(initial, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}
	runTestGit(t, root, "add", "notes.txt")
	runTestGit(t, root, "commit", "-m", "initial")

	modified := append([]string{}, initial...)
	modified[1] = "changed 2"
	modified[24] = "changed 25"
	if err := os.WriteFile(path, []byte(strings.Join(modified, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write modified file: %v", err)
	}

	service := Service{workspaceRoot: func() string { return root }}
	preview, err := service.ApplyHunkAction(HunkActionRequest{
		Path:      "notes.txt",
		Action:    gitHunkActionStage,
		DiffKind:  gitDiffKindUnstaged,
		HunkIndex: 1,
	})
	if err != nil {
		t.Fatalf("ApplyHunkAction returned error: %v", err)
	}
	if preview.Status.Available != true {
		t.Fatalf("expected status in hunk action result: %#v", preview)
	}
	staged := runTestGitOutput(t, root, "diff", "--cached", "--", "notes.txt")
	if !strings.Contains(staged, "changed 2") {
		t.Fatalf("expected first hunk to be staged: %q", staged)
	}
	if strings.Contains(staged, "changed 25") {
		t.Fatalf("expected second hunk to remain unstaged: %q", staged)
	}
}

func runTestGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}

func runTestGitOutput(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
	return string(output)
}
