package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommitChangesCreatesCommitFromStagedChangesOnly(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "nexus@example.test")
	runGit(t, root, "config", "user.name", "Nexus")
	writeGitTestFile(t, root, "app.go", "package main\n")
	runGit(t, root, "add", "--", "app.go")
	runGit(t, root, "commit", "-m", "initial")

	writeGitTestFile(t, root, "app.go", "package main\n\nfunc main() {}\n")
	writeGitTestFile(t, root, "draft.txt", "not staged\n")
	runGit(t, root, "add", "--", "app.go")

	result, err := New().CommitChanges(root, "Add main", "Keep draft unstaged.")
	if err != nil {
		t.Fatalf("CommitChanges returned error: %v", err)
	}
	if result.Hash == "" || result.ShortHash == "" || result.Subject != "Add main" {
		t.Fatalf("unexpected commit result: %#v", result)
	}
	if !strings.Contains(result.StagedStat, "app.go") || strings.Contains(result.StagedStat, "draft.txt") {
		t.Fatalf("unexpected staged stat: %q", result.StagedStat)
	}
	if len(result.Status.StagedFiles) != 0 || len(result.Status.UnstagedFiles) != 1 {
		t.Fatalf("expected unstaged draft to remain after commit, got %#v", result.Status)
	}
	log := mustGitOutput(root, "log", "-1", "--pretty=%s%n%b")
	if !strings.Contains(log, "Add main") || !strings.Contains(log, "Keep draft unstaged.") {
		t.Fatalf("commit message/body were not persisted:\n%s", log)
	}
}

func TestCommitChangesRequiresMessageAndStagedChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "nexus@example.test")
	runGit(t, root, "config", "user.name", "Nexus")
	writeGitTestFile(t, root, "app.go", "package main\n")
	runGit(t, root, "add", "--", "app.go")
	runGit(t, root, "commit", "-m", "initial")

	blank, err := New().CommitChanges(root, "   ", "")
	if err != nil {
		t.Fatalf("blank CommitChanges returned error: %v", err)
	}
	if blank.Message != "Commit message is required." {
		t.Fatalf("expected message validation, got %#v", blank)
	}

	empty, err := New().CommitChanges(root, "No changes", "")
	if err != nil {
		t.Fatalf("empty CommitChanges returned error: %v", err)
	}
	if empty.Message != "No staged changes are available to commit." {
		t.Fatalf("expected staged-change validation, got %#v", empty)
	}
}

func writeGitTestFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
