package git

import (
	"os/exec"
	"strings"
	"testing"
)

func TestCreateBranchCreatesBranchWithoutCheckoutByDefault(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	initGitRepoWithCommit(t, root)

	result, err := New().CreateBranch(root, "feature/native-tools", "", false)
	if err != nil {
		t.Fatalf("CreateBranch returned error: %v", err)
	}
	if result.BranchName != "feature/native-tools" || result.CheckedOut {
		t.Fatalf("unexpected branch result: %#v", result)
	}
	if !strings.Contains(result.Message, "Created branch feature/native-tools") {
		t.Fatalf("unexpected branch message: %q", result.Message)
	}
	if result.Status.Branch != "master" && result.Status.Branch != "main" {
		t.Fatalf("expected current branch to remain unchanged, got %#v", result.Status)
	}
	if out := mustGitOutput(root, "branch", "--list", "feature/native-tools"); !strings.Contains(out, "feature/native-tools") {
		t.Fatalf("created branch was not listed: %q", out)
	}
}

func TestCreateBranchCanCheckoutAndRejectsInvalidOrExistingBranches(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable is not available")
	}
	root := t.TempDir()
	initGitRepoWithCommit(t, root)

	checkedOut, err := New().CreateBranch(root, "feature/checked-out", "HEAD", true)
	if err != nil {
		t.Fatalf("CreateBranch checkout returned error: %v", err)
	}
	if !checkedOut.CheckedOut || checkedOut.Status.Branch != "feature/checked-out" {
		t.Fatalf("expected checkout branch status, got %#v", checkedOut)
	}

	duplicate, err := New().CreateBranch(root, "feature/checked-out", "", false)
	if err != nil {
		t.Fatalf("duplicate CreateBranch returned error: %v", err)
	}
	if duplicate.Message != "Git branch already exists." {
		t.Fatalf("expected duplicate rejection, got %#v", duplicate)
	}

	invalid, err := New().CreateBranch(root, "-danger", "", false)
	if err != nil {
		t.Fatalf("invalid CreateBranch returned error: %v", err)
	}
	if invalid.Message != "Git branch names and start points must not start with '-'." {
		t.Fatalf("expected invalid branch rejection, got %#v", invalid)
	}
}

func initGitRepoWithCommit(t *testing.T, root string) {
	t.Helper()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "nexus@example.test")
	runGit(t, root, "config", "user.name", "Nexus")
	writeGitTestFile(t, root, "README.md", "# Repo\n")
	runGit(t, root, "add", "--", "README.md")
	runGit(t, root, "commit", "-m", "initial")
}
