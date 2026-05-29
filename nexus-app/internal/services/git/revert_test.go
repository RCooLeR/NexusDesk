package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanRevertChangesPreparesTrackedWorktreeRestore(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	initGitRepoWithCommit(t, root)
	writeGitTestFile(t, root, "README.md", "# Repo\n\nchanged\n")

	plan, err := New().PlanRevertChanges(root, "README.md", "")
	if err != nil {
		t.Fatalf("PlanRevertChanges returned error: %v", err)
	}
	if plan.Action != RevertActionWrite || plan.Scope != RevertScopeWorktree {
		t.Fatalf("unexpected plan action/scope: %#v", plan)
	}
	if plan.Content != "# Repo\n" {
		t.Fatalf("expected HEAD content, got %q", plan.Content)
	}
	if !strings.Contains(plan.Message, "Prepared to restore README.md") {
		t.Fatalf("unexpected message: %q", plan.Message)
	}
}

func TestPlanRevertChangesRequiresExplicitUntrackedScope(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	initGitRepoWithCommit(t, root)
	writeGitTestFile(t, root, "scratch.txt", "draft\n")

	blocked, err := New().PlanRevertChanges(root, "scratch.txt", "")
	if err != nil {
		t.Fatalf("PlanRevertChanges returned error: %v", err)
	}
	if blocked.Action != "" || !strings.Contains(blocked.Message, "scope=untracked") {
		t.Fatalf("expected explicit untracked scope requirement, got %#v", blocked)
	}
	plan, err := New().PlanRevertChanges(root, "scratch.txt", "untracked")
	if err != nil {
		t.Fatalf("PlanRevertChanges returned error: %v", err)
	}
	if plan.Action != RevertActionDelete || plan.Scope != RevertScopeUntracked {
		t.Fatalf("expected delete plan, got %#v", plan)
	}
}

func TestPlanRevertChangesRejectsStagedChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	initGitRepoWithCommit(t, root)
	writeGitTestFile(t, root, "README.md", "# Repo\n\nstaged\n")
	runGit(t, root, "add", "--", "README.md")

	plan, err := New().PlanRevertChanges(root, "README.md", "")
	if err != nil {
		t.Fatalf("PlanRevertChanges returned error: %v", err)
	}
	if plan.Action != "" || !strings.Contains(plan.Message, "Staged changes are not reverted") {
		t.Fatalf("expected staged rejection, got %#v", plan)
	}
}

func TestPlanRevertChangesRejectsBinaryHeadContent(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "nexus@example.test")
	runGit(t, root, "config", "user.name", "Nexus")
	path := filepath.Join(root, "blob.bin")
	if err := os.WriteFile(path, []byte{'a', 0x00, 'b'}, 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "--", "blob.bin")
	runGit(t, root, "commit", "-m", "add binary")
	if err := os.WriteFile(path, []byte{'c', 0x00, 'd'}, 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := New().PlanRevertChanges(root, "blob.bin", "")
	if err != nil {
		t.Fatalf("PlanRevertChanges returned error: %v", err)
	}
	if plan.Action != "" || !strings.Contains(plan.Message, "Binary Git content") {
		t.Fatalf("expected binary rejection, got %#v", plan)
	}
}
